import React, { useState } from "react";
import { PageLayout } from "../../components/ui";

const DEFAULT_BASE = "C:\\WoW\\Octo";

// Each importer maps to an App binding that takes the client base folder.
const IMPORTS = [
  {
    id: "cache",
    name: "WDB Cache",
    fn: "RunCacheImport",
    sub: "WDB\\*.wdb",
    desc:
      "Patch item / quest / creature / gameobject data from your client's WDB caches — everything you've queried in-game. Overlays the freshest server values; existing data is never wiped.",
  },
  {
    id: "maps",
    name: "Zone Maps",
    fn: "RunMapImport",
    sub: "Data\\*.MPQ (or loose BlizzardInterfaceArt\\WorldMap)",
    desc:
      "Generate fully-revealed zone maps from the client world-map art into data/maps. Read straight from the client's MPQ archives (in memory) when present. These power the map in the NPC view (kept local, never shipped).",
  },
  {
    id: "icons",
    name: "Icons",
    fn: "RunIconImport",
    sub: "Data\\*.MPQ (or loose BlizzardInterfaceArt\\Icons)",
    desc:
      "Decode item/spell/ability icons from the client art into data/icons. Read straight from the client's MPQ archives (in memory) when present. Resolves icons locally without downloading from the web.",
  },
  {
    id: "dbc",
    name: "DBC Reference Data",
    fn: "RunDbcImport",
    sub: "Data\\*.MPQ (or loose DBFilesClient\\*.dbc)",
    desc:
      "Regenerate reference data from the client DBCs (zones, skills, quest sorts, factions, item sets, icons, spell text) and re-apply it to the database. Read straight from the client's MPQ archives (in memory) when present. Does not touch creature/item/quest templates.",
  },
];

function ToolsPage({ onNavigate }) {
  const [base, setBase] = useState(
    () => localStorage.getItem("toolsBasePath") || DEFAULT_BASE
  );
  const [running, setRunning] = useState(null);
  const [reports, setReports] = useState({});
  const [whatsNew, setWhatsNew] = useState(null);
  const [wnLoading, setWnLoading] = useState(false);

  const loadWhatsNew = async () => {
    const app = window?.go?.main?.App;
    if (!app?.WhatsNew) {
      setWhatsNew({ error: "Binding not found (dev build?)" });
      return;
    }
    setWnLoading(true);
    try {
      setWhatsNew(await app.WhatsNew());
    } catch (e) {
      setWhatsNew({ error: String(e) });
    } finally {
      setWnLoading(false);
    }
  };

  const saveBase = (v) => {
    setBase(v);
    localStorage.setItem("toolsBasePath", v);
  };

  const run = async (imp) => {
    const app = window?.go?.main?.App;
    if (!app || !app[imp.fn]) {
      setReports((r) => ({
        ...r,
        [imp.id]: { success: false, title: "Unavailable", lines: ["Binding not found (dev build?)"] },
      }));
      return;
    }
    setRunning(imp.id);
    try {
      const rep = await app[imp.fn](base);
      setReports((r) => ({ ...r, [imp.id]: rep }));
    } catch (e) {
      setReports((r) => ({
        ...r,
        [imp.id]: { success: false, title: "Failed", lines: [String(e)] },
      }));
    } finally {
      setRunning(null);
    }
  };

  return (
    <PageLayout>
      <div className="max-w-3xl mx-auto p-6 space-y-6 overflow-y-auto h-full">
        <div>
          <h2 className="text-xl text-wow-gold font-bold mb-1">Data Tools</h2>
          <p className="text-gray-400 text-sm">
            Refresh InkLab's data from your local WoW client. Nothing is uploaded — each
            import reads the files under the folder below.
          </p>
        </div>

        <div className="bg-gray-800/50 border border-gray-700/50 rounded-xl p-4">
          <label className="block text-[11px] uppercase font-bold text-gray-500 mb-1">
            WoW client folder
          </label>
          <input
            value={base}
            onChange={(e) => saveBase(e.target.value)}
            spellCheck={false}
            className="w-full bg-bg-dark border border-border-light rounded px-3 py-2 font-mono text-sm text-gray-200 focus:outline-none focus:border-wow-gold/50"
            placeholder={DEFAULT_BASE}
          />
          <p className="text-[11px] text-gray-600 mt-1">
            Reads <span className="font-mono">Data\*.MPQ</span> directly when present
            (nothing is written back), plus <span className="font-mono">WDB\</span> for
            the cache import; falls back to loose{" "}
            <span className="font-mono">DBFilesClient\</span> /{" "}
            <span className="font-mono">BlizzardInterfaceArt\</span> folders.
          </p>
        </div>

        {/* What's New — diff of the live DB vs the last committed baseline */}
        <div className="bg-gray-800/50 border border-gray-700/50 rounded-xl p-4">
          <div className="flex items-start justify-between gap-4">
            <div className="min-w-0">
              <h3 className="text-white font-semibold">What's New</h3>
              <p className="text-gray-400 text-sm mt-1">
                Rows added or changed in the database since the last committed
                baseline — e.g. items, NPCs and objects your imports pulled in.
                Click an entry to open it.
              </p>
              {whatsNew?.baseline && (
                <p className="text-[11px] text-gray-600 mt-1">vs {whatsNew.baseline}</p>
              )}
            </div>
            <button
              onClick={loadWhatsNew}
              disabled={wnLoading}
              className="shrink-0 bg-wow-gold/90 hover:bg-wow-gold text-black font-bold px-5 py-2 rounded transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {wnLoading ? "Checking…" : "Check"}
            </button>
          </div>

          {whatsNew?.error && (
            <div className="mt-3 rounded border border-red-500/30 bg-red-500/5 p-3 text-red-400 text-sm">
              {whatsNew.error}
            </div>
          )}

          {whatsNew && !whatsNew.error && (
            <div className="mt-3 space-y-3">
              {!whatsNew.groups?.length && (
                <div className="text-gray-500 text-sm italic">
                  No changes since the baseline.
                </div>
              )}
              {whatsNew.groups?.map((g) => (
                <div key={g.type} className="rounded border border-gray-700/50 bg-black/20 p-3">
                  <div className="text-sm font-bold text-gray-200 mb-2">
                    {g.label}{" "}
                    <span className="text-green-400 font-normal">+{g.added} added</span>
                    {g.changed > 0 && (
                      <span className="text-blue-400 font-normal"> • {g.changed} changed</span>
                    )}
                  </div>
                  <div className="flex flex-wrap gap-1.5">
                    {g.entries?.map((e) => (
                      <button
                        key={`${e.type}-${e.id}`}
                        onClick={() => onNavigate?.(e.type, e.id)}
                        title={`${e.change} — open ${e.type} ${e.id}`}
                        className={`px-2 py-1 rounded text-xs border transition-colors text-left ${
                          e.change === "added"
                            ? "border-green-600/40 bg-green-600/10 hover:bg-green-600/20 text-green-200"
                            : "border-blue-600/40 bg-blue-600/10 hover:bg-blue-600/20 text-blue-200"
                        }`}
                      >
                        <span className="text-gray-500 font-mono">[{e.id}]</span>{" "}
                        {e.name || "(unnamed)"}
                      </button>
                    ))}
                    {(g.added + g.changed) > g.entries.length && (
                      <span className="text-gray-600 text-xs self-center">
                        … {g.added + g.changed - g.entries.length} more
                      </span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {IMPORTS.map((imp) => {
          const rep = reports[imp.id];
          const busy = running === imp.id;
          return (
            <div key={imp.id} className="bg-gray-800/50 border border-gray-700/50 rounded-xl p-4">
              <div className="flex items-start justify-between gap-4">
                <div className="min-w-0">
                  <h3 className="text-white font-semibold">{imp.name}</h3>
                  <p className="text-gray-400 text-sm mt-1">{imp.desc}</p>
                  <p className="text-[11px] text-gray-600 font-mono mt-1">{imp.sub}</p>
                </div>
                <button
                  onClick={() => run(imp)}
                  disabled={!!running}
                  className="shrink-0 bg-wow-gold/90 hover:bg-wow-gold text-black font-bold px-5 py-2 rounded transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  {busy ? "Running…" : "Run"}
                </button>
              </div>
              {rep && (
                <div
                  className={`mt-3 rounded border p-3 ${
                    rep.success
                      ? "border-green-500/30 bg-green-500/5"
                      : "border-red-500/30 bg-red-500/5"
                  }`}
                >
                  <div className={`font-bold text-sm ${rep.success ? "text-green-400" : "text-red-400"}`}>
                    {rep.title}
                  </div>
                  {rep.lines?.map((l, i) => (
                    <div key={i} className="text-gray-300 font-mono text-xs mt-0.5 break-all">
                      {l}
                    </div>
                  ))}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </PageLayout>
  );
}

export default ToolsPage;
