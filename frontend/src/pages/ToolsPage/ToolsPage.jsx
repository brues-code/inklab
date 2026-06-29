import React, { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { PageLayout } from '../../components/ui'
import { useDataStatus, useWhatsNew, whatsNewQuery } from '../../hooks/queries/app'
import { DEFAULT_WOW_BASE } from '../../utils/constants'
import { useEntityNavigate } from '../../utils/entityNav'

const DEFAULT_BASE = DEFAULT_WOW_BASE

// Each importer maps to an App binding that takes the client base folder.
const IMPORTS = [
    {
        id: 'client',
        name: 'Client Data (icons, maps, DBC)',
        fn: 'RunClientImport',
        sub: 'Data\\*.MPQ (or loose DBFilesClient\\ + BlizzardInterfaceArt\\)',
        desc: "One pass over your WoW client: decode icons → data/icons, build fully-revealed zone maps → data/maps, and regenerate reference data (zones, skills, quest sorts, factions, item sets, spell text) into the database. Reads straight from the client's MPQ archives in memory when present (nothing is written back).",
    },
    {
        id: 'cache',
        name: 'WDB Cache',
        fn: 'RunCacheImport',
        sub: 'WDB\\*.wdb',
        desc: "Patch item / quest / creature / gameobject data from your client's WDB caches — everything you've queried in-game. Overlays the freshest server values; existing data is never wiped.",
    },
    {
        id: 'spawnZones',
        name: 'Rebuild Spawn Zones',
        fn: 'RebuildSpawnZones',
        sub: 'data/area_grid.bin + world DB coordinates',
        desc: 'Re-resolve every creature and gameobject spawn to the correct zone using the client area grid (the actual terrain), not overlapping zone boxes. Fixes border mislabels — e.g. Westfall mobs counted in Elwynn, or Elwynn mobs swallowed by Duskwood. No octowow scraping; reports the per-zone net change. Run after a Client Data import so the area grid exists.',
    },
]

// DatasetInventory lists every client-derived dataset and whether it's present,
// so the user can see exactly what's missing — and which client file feeds it.
// Missing datasets (count 0) sort first and name their source DBC / art folder.
function DatasetInventory({ datasets }) {
    const sorted = [...datasets].sort(
        (a, b) => (a.count > 0) - (b.count > 0) || a.label.localeCompare(b.label),
    )
    const missing = datasets.filter((d) => d.count === 0).length
    return (
        <div className="mt-3 border-t border-gray-700/50 pt-3">
            <div className="mb-2 flex items-center justify-between">
                <span className="text-[11px] font-bold uppercase text-gray-500">
                    Data inventory
                </span>
                <span
                    className={`font-mono text-[11px] ${missing > 0 ? 'text-red-400' : 'text-green-400'}`}
                >
                    {missing > 0 ? `${missing} missing` : 'all present'}
                </span>
            </div>
            <div className="grid grid-cols-1 gap-1.5 sm:grid-cols-2">
                {sorted.map((d) => {
                    const ok = d.count > 0
                    return (
                        <div
                            key={d.key}
                            className={`flex items-center gap-2 rounded border px-2 py-1 ${
                                ok
                                    ? 'border-gray-700/40 bg-black/20'
                                    : 'border-red-500/40 bg-red-500/10'
                            }`}
                        >
                            <span
                                className={`h-1.5 w-1.5 shrink-0 rounded-full ${ok ? 'bg-green-500' : 'bg-red-500'}`}
                            />
                            <div className="min-w-0 flex-1">
                                <div className="flex items-center justify-between gap-2">
                                    <span
                                        className={`truncate text-xs ${ok ? 'text-gray-200' : 'font-semibold text-red-300'}`}
                                    >
                                        {d.label}
                                    </span>
                                    <span
                                        className={`shrink-0 font-mono text-[11px] ${ok ? 'text-gray-400' : 'text-red-400'}`}
                                    >
                                        {ok ? d.count.toLocaleString() : 'missing'}
                                    </span>
                                </div>
                                <div
                                    className="truncate font-mono text-[10px] text-gray-600"
                                    title={d.source}
                                >
                                    {d.source}
                                </div>
                            </div>
                        </div>
                    )
                })}
            </div>
        </div>
    )
}

function ToolsPage() {
    const entityNavigate = useEntityNavigate()
    const queryClient = useQueryClient()
    const [base, setBase] = useState(() => localStorage.getItem('toolsBasePath') || DEFAULT_BASE)
    const [running, setRunning] = useState(null)
    const [reports, setReports] = useState({})
    const { data: status } = useDataStatus()

    const { data: whatsNew, isFetching: wnLoading } = useWhatsNew()
    // staleTime: 0 forces a fresh diff on every Check; the cached result still
    // persists across navigation via the hook's Infinity staleTime.
    const loadWhatsNew = () => queryClient.fetchQuery({ ...whatsNewQuery, staleTime: 0 })

    const saveBase = (v) => {
        setBase(v)
        localStorage.setItem('toolsBasePath', v)
    }

    const run = async (imp) => {
        const app = window?.go?.main?.App
        if (!app || !app[imp.fn]) {
            setReports((r) => ({
                ...r,
                [imp.id]: {
                    success: false,
                    title: 'Unavailable',
                    lines: ['Binding not found (dev build?)'],
                },
            }))
            return
        }
        setRunning(imp.id)
        try {
            const rep = await app[imp.fn](base)
            setReports((r) => ({ ...r, [imp.id]: rep }))
        } catch (e) {
            setReports((r) => ({
                ...r,
                [imp.id]: { success: false, title: 'Failed', lines: [String(e)] },
            }))
        } finally {
            setRunning(null)
            // An import rewrites reference data, icons and maps; drop every cached
            // query so views refetch the fresh data (overrides staleTime: Infinity).
            // This includes dataStatus, refreshing the inventory below.
            queryClient.invalidateQueries()
        }
    }

    // Categories that are populated by an import and worth warning about when empty.
    const missing = status
        ? [status.icons === 0 && 'icons', status.maps === 0 && 'zone maps'].filter(Boolean)
        : []

    return (
        <PageLayout>
            <div className="mx-auto h-full max-w-3xl space-y-6 overflow-y-auto p-6">
                <div>
                    <h2 className="mb-1 text-xl font-bold text-wow-gold">Import</h2>
                    <p className="text-sm text-gray-400">
                        Refresh InkLab's data from your local WoW client. Nothing is uploaded — each
                        import reads the files under the folder below.
                    </p>
                </div>

                {missing.length > 0 && (
                    <div className="rounded-xl border border-amber-500/40 bg-amber-500/10 p-4">
                        <div className="text-sm font-semibold text-amber-300">
                            ⚠️ No {missing.join(' or ')} found
                        </div>
                        <p className="mt-1 text-sm text-amber-200/80">
                            InkLab ships without bundled {missing.join(' / ')} — they're built from
                            your local WoW client. Set the client folder below and run the matching
                            import, or items will show a placeholder icon and NPCs won't show a zone
                            map.
                        </p>
                    </div>
                )}

                <div className="rounded-xl border border-gray-700/50 bg-gray-800/50 p-4">
                    <label className="mb-1 block text-[11px] font-bold uppercase text-gray-500">
                        WoW client folder
                    </label>
                    <input
                        value={base}
                        onChange={(e) => saveBase(e.target.value)}
                        spellCheck={false}
                        className="w-full rounded border border-border-light bg-bg-dark px-3 py-2 font-mono text-sm text-gray-200 focus:border-wow-gold/50 focus:outline-none"
                        placeholder={DEFAULT_BASE}
                    />
                    <p className="mt-1 text-[11px] text-gray-600">
                        Reads <span className="font-mono">Data\*.MPQ</span> directly when present
                        (nothing is written back), plus <span className="font-mono">WDB\</span> for
                        the cache import; falls back to loose{' '}
                        <span className="font-mono">DBFilesClient\</span> /{' '}
                        <span className="font-mono">BlizzardInterfaceArt\</span> folders.
                    </p>
                </div>

                {IMPORTS.map((imp) => {
                    const rep = reports[imp.id]
                    const busy = running === imp.id
                    return (
                        <div
                            key={imp.id}
                            className="rounded-xl border border-gray-700/50 bg-gray-800/50 p-4"
                        >
                            <div className="flex items-start justify-between gap-4">
                                <div className="min-w-0">
                                    <h3 className="font-semibold text-white">{imp.name}</h3>
                                    <p className="mt-1 text-sm text-gray-400">{imp.desc}</p>
                                    <p className="mt-1 font-mono text-[11px] text-gray-600">
                                        {imp.sub}
                                    </p>
                                </div>
                                <button
                                    onClick={() => run(imp)}
                                    disabled={!!running}
                                    className="shrink-0 rounded bg-wow-gold/90 px-5 py-2 font-bold text-black transition-colors hover:bg-wow-gold disabled:cursor-not-allowed disabled:opacity-40"
                                >
                                    {busy ? 'Running…' : 'Run'}
                                </button>
                            </div>
                            {imp.id === 'client' && status?.datasets?.length > 0 && (
                                <DatasetInventory datasets={status.datasets} />
                            )}
                            {rep && (
                                <div
                                    className={`mt-3 rounded border p-3 ${
                                        rep.success
                                            ? 'border-green-500/30 bg-green-500/5'
                                            : 'border-red-500/30 bg-red-500/5'
                                    }`}
                                >
                                    <div
                                        className={`text-sm font-bold ${rep.success ? 'text-green-400' : 'text-red-400'}`}
                                    >
                                        {rep.title}
                                    </div>
                                    {rep.lines?.map((l, i) => (
                                        <div
                                            key={i}
                                            className="mt-0.5 break-all font-mono text-xs text-gray-300"
                                        >
                                            {l}
                                        </div>
                                    ))}
                                </div>
                            )}
                        </div>
                    )
                })}

                {/* What's New — diff of the live DB vs the baseline. Placed last: it's
            noise for a brand-new user who hasn't imported anything yet. */}
                <div className="rounded-xl border border-gray-700/50 bg-gray-800/50 p-4">
                    <div className="flex items-start justify-between gap-4">
                        <div className="min-w-0">
                            <h3 className="font-semibold text-white">What's New</h3>
                            <p className="mt-1 text-sm text-gray-400">
                                Rows added or changed in the database since the last committed
                                baseline — e.g. items, NPCs and objects your imports pulled in.
                                Click an entry to open it.
                            </p>
                            {whatsNew?.baseline && (
                                <p className="mt-1 text-[11px] text-gray-600">
                                    vs {whatsNew.baseline}
                                </p>
                            )}
                        </div>
                        <button
                            onClick={() => loadWhatsNew()}
                            disabled={wnLoading}
                            className="shrink-0 rounded bg-wow-gold/90 px-5 py-2 font-bold text-black transition-colors hover:bg-wow-gold disabled:cursor-not-allowed disabled:opacity-40"
                        >
                            {wnLoading ? 'Checking…' : 'Check'}
                        </button>
                    </div>

                    {whatsNew?.error && (
                        <div className="mt-3 rounded border border-red-500/30 bg-red-500/5 p-3 text-sm text-red-400">
                            {whatsNew.error}
                        </div>
                    )}

                    {whatsNew && !whatsNew.error && (
                        <div className="mt-3 space-y-3">
                            {!whatsNew.groups?.length && (
                                <div className="text-sm italic text-gray-500">
                                    No changes since the baseline.
                                </div>
                            )}
                            {whatsNew.groups?.map((g) => (
                                <div
                                    key={g.type}
                                    className="rounded border border-gray-700/50 bg-black/20 p-3"
                                >
                                    <div className="mb-2 text-sm font-bold text-gray-200">
                                        {g.label}{' '}
                                        <span className="font-normal text-green-400">
                                            +{g.added} added
                                        </span>
                                        {g.changed > 0 && (
                                            <span className="font-normal text-blue-400">
                                                {' '}
                                                • {g.changed} changed
                                            </span>
                                        )}
                                    </div>
                                    <div className="flex flex-wrap gap-1.5">
                                        {g.entries?.map((e) => (
                                            <button
                                                key={`${e.type}-${e.id}`}
                                                onClick={() => entityNavigate(e.type, e.id)}
                                                title={`${e.change} — open ${e.type} ${e.id}`}
                                                className={`rounded border px-2 py-1 text-left text-xs transition-colors ${
                                                    e.change === 'added'
                                                        ? 'border-green-600/40 bg-green-600/10 text-green-200 hover:bg-green-600/20'
                                                        : 'border-blue-600/40 bg-blue-600/10 text-blue-200 hover:bg-blue-600/20'
                                                }`}
                                            >
                                                <span className="font-mono text-gray-500">
                                                    [{e.id}]
                                                </span>{' '}
                                                {e.name || '(unnamed)'}
                                            </button>
                                        ))}
                                        {g.added + g.changed > g.entries.length && (
                                            <span className="self-center text-xs text-gray-600">
                                                … {g.added + g.changed - g.entries.length} more
                                            </span>
                                        )}
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            </div>
        </PageLayout>
    )
}

export default ToolsPage
