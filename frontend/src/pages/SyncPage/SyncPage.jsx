import React, { useState, useEffect } from 'react'
import {
    FullSyncNpcs,
    FullSyncItems,
    FullSyncQuests,
    FullSyncObjects,
    StopSync,
} from '../../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime'
import { queryClient } from '../../queryClient'
import { queryKeys } from '../../hooks/queries/keys'
import { useSyncStats } from '../../hooks/queries/app'
import { PageLayout } from '../../components/ui'

const SYNC_TYPES = [
    { id: 'npc', name: 'NPCs', icon: '👤' },
    { id: 'item', name: 'Items', icon: '⚔️' },
    { id: 'quest', name: 'Quests', icon: '📜' },
    { id: 'object', name: 'Objects', icon: '📦' },
]

function SyncPage() {
    // Global stats
    const { data: syncStats } = useSyncStats()

    // Selection
    const [activeSyncType, setActiveSyncType] = useState(() => {
        return localStorage.getItem('lastActiveSyncType') || 'npc'
    })

    // Start IDs (linked to the input field)
    const [startIds, setStartIds] = useState({
        npc: parseInt(localStorage.getItem('lastSyncedNpcId') || '0', 10),
        item: parseInt(localStorage.getItem('lastSyncedItemId') || '0', 10),
        quest: parseInt(localStorage.getItem('lastSyncedQuestId') || '0', 10),
        object: parseInt(localStorage.getItem('lastSyncedObjectId') || '0', 10),
        model: parseInt(localStorage.getItem('lastSyncedModelId') || '0', 10),
    })

    // Sync state
    const [syncing, setSyncing] = useState(false)
    const [syncResult, setSyncResult] = useState(null)
    const [syncProgress, setSyncProgress] = useState(null)
    const [syncLog, setSyncLog] = useState([])

    useEffect(() => {
        // NPC Progress
        EventsOn('sync:npc_full:progress', (data) => handleProgress('npc', data))
        EventsOn('sync:npc_full:error', (msg) => handleSyncError('npc', msg))
        EventsOn('sync:npc_full:complete', (msg) => handleSyncDone('npc', msg))

        // Item Progress
        EventsOn('sync:progress', (data) => handleProgress('item', data))
        EventsOn('sync:item_full:error', (msg) => handleSyncError('item', msg))
        EventsOn('sync:item_full:complete', (msg) => handleSyncDone('item', msg))

        // Quest Progress
        EventsOn('sync:quests:progress', (data) => handleProgress('quest', data))
        EventsOn('sync:quests_full:complete', (msg) => handleSyncDone('quest', msg))

        // Object Progress
        EventsOn('sync:objects:progress', (data) => handleProgress('object', data))
        EventsOn('sync:objects_full:error', (msg) => handleSyncError('object', msg))
        EventsOn('sync:objects_full:complete', (msg) => handleSyncDone('object', msg))

        return () => {
            EventsOff('sync:npc_full:progress')
            EventsOff('sync:npc_full:error')
            EventsOff('sync:npc_full:complete')
            EventsOff('sync:progress')
            EventsOff('sync:item_full:error')
            EventsOff('sync:item_full:complete')
            EventsOff('sync:quests:progress')
            EventsOff('sync:quests_full:complete')
            EventsOff('sync:objects:progress')
            EventsOff('sync:objects_full:error')
            EventsOff('sync:objects_full:complete')
        }
    }, [])

    const handleProgress = (type, data) => {
        setSyncProgress({
            type,
            current: data.current,
            total: data.total,
            id: data.itemId || data.id,
            name: data.itemName || `${type.toUpperCase()} ID ${data.itemId || data.id}`,
        })

        // Update startIds for resume next time
        const id = data.itemId || data.id
        setStartIds((prev) => ({ ...prev, [type]: id }))

        // Persist to localStorage
        const storageKey = `lastSynced${type.charAt(0).toUpperCase() + type.slice(1)}Id`
        localStorage.setItem(storageKey, id.toString())

        setSyncLog((prev) => {
            if (prev.length > 0 && prev[0].id === id) return prev
            return [{ id, name: data.itemName || `${type.toUpperCase()} ID ${id}` }, ...prev].slice(
                0,
                5,
            )
        })
    }

    const handleSyncError = (type, msg) => {
        setSyncing(false)
        setSyncResult({ type, error: msg })
        queryClient.invalidateQueries({ queryKey: queryKeys.syncStats })
    }

    const handleSyncDone = (type, msg) => {
        setSyncing(false)
        setSyncResult({ type, message: msg || 'Sync complete!' })
        // A full sync rewrites many rows; drop the cache so every view refetches
        // (this includes syncStats).
        queryClient.invalidateQueries()
    }

    const handleStartSync = async () => {
        if (syncing) return

        setSyncing(true)
        setSyncResult(null)
        setSyncLog([])

        const startId = startIds[activeSyncType]
        localStorage.setItem('lastActiveSyncType', activeSyncType)

        try {
            let result
            switch (activeSyncType) {
                case 'npc':
                    await FullSyncNpcs(startId, 100)
                    break
                case 'item':
                    await FullSyncItems(100, true, startId)
                    break
                case 'quest':
                    await FullSyncQuests(100, startId)
                    break
                case 'object':
                    await FullSyncObjects(100, startId)
                    break
            }
        } catch (error) {
            handleSyncError(activeSyncType, error.toString())
        }
    }

    const handleStopSync = async () => {
        try {
            await StopSync()
            setSyncing(false)
            setSyncResult({
                type: activeSyncType,
                message: 'Sync stop requested. It will pause after the current item finishes.',
            })
        } catch (e) {
            console.error(e)
        }
    }

    const handleResetProgress = (type) => {
        if (window.confirm(`Reset progress for ${type.toUpperCase()}?`)) {
            const storageKey = `lastSynced${type.charAt(0).toUpperCase() + type.slice(1)}Id`
            localStorage.removeItem(storageKey)
            setStartIds((prev) => ({ ...prev, [type]: 0 }))
        }
    }

    return (
        <PageLayout>
            <div className="mx-auto w-full max-w-4xl flex-1 overflow-y-auto p-8">
                <h1 className="mb-8 text-3xl font-bold text-white">Data Synchronization</h1>

                {/* Global Stats */}
                {syncStats && (
                    <div className="mb-8 grid grid-cols-2 gap-4 md:grid-cols-4">
                        <div className="rounded-xl border border-gray-700/50 bg-gray-800/50 p-4">
                            <div className="mb-1 text-[10px] font-bold uppercase text-gray-500">
                                NPCs
                            </div>
                            <div className="font-mono text-xl text-wow-gold">
                                {syncStats.creatureCount}
                            </div>
                        </div>
                        <div className="rounded-xl border border-gray-700/50 bg-gray-800/50 p-4">
                            <div className="mb-1 text-[10px] font-bold uppercase text-gray-500">
                                Items
                            </div>
                            <div className="font-mono text-xl text-wow-gold">
                                {syncStats.itemCount}
                            </div>
                        </div>
                        <div className="rounded-xl border border-gray-700/50 bg-gray-800/50 p-4">
                            <div className="mb-1 text-[10px] font-bold uppercase text-gray-500">
                                Quests
                            </div>
                            <div className="font-mono text-xl text-wow-gold">
                                {syncStats.questCount}
                            </div>
                        </div>
                        <div className="rounded-xl border border-gray-700/50 bg-gray-800/50 p-4">
                            <div className="mb-1 text-[10px] font-bold uppercase text-gray-500">
                                Max Item ID
                            </div>
                            <div className="font-mono text-xl text-gray-400">
                                {syncStats.maxItemID}
                            </div>
                        </div>
                    </div>
                )}

                <div className="rounded-2xl border border-gray-700 bg-gray-800/40 p-8 shadow-2xl backdrop-blur-sm">
                    <div className="mb-8 flex flex-col justify-between gap-6 md:flex-row md:items-center">
                        <div>
                            <h2 className="mb-2 text-2xl font-bold text-white">Sync Engine</h2>
                            <p className="text-sm text-gray-400">
                                Download and update database from Web & MySQL sources.
                            </p>
                        </div>

                        {/* Type Switcher */}
                        <div className="flex rounded-xl border border-white/5 bg-black/40 p-1">
                            {SYNC_TYPES.map((type) => (
                                <button
                                    key={type.id}
                                    disabled={syncing}
                                    onClick={() => setActiveSyncType(type.id)}
                                    className={`rounded-lg px-4 py-2 text-sm font-bold transition-all ${
                                        activeSyncType === type.id
                                            ? 'bg-wow-gold text-gray-900 shadow-lg'
                                            : 'text-gray-400 hover:text-white'
                                    } disabled:opacity-50`}
                                >
                                    <span className="mr-2">{type.icon}</span>
                                    {type.name}
                                </button>
                            ))}
                        </div>
                    </div>

                    {/* Sync Configuration */}
                    <div className="mb-8 grid grid-cols-1 gap-8 md:grid-cols-2">
                        <div className="space-y-4">
                            <label className="block">
                                <span className="mb-2 block text-xs font-bold uppercase text-gray-500">
                                    Starting Entry ID
                                </span>
                                <div className="group relative">
                                    <input
                                        type="number"
                                        disabled={syncing}
                                        value={startIds[activeSyncType]}
                                        onChange={(e) =>
                                            setStartIds((prev) => ({
                                                ...prev,
                                                [activeSyncType]: parseInt(e.target.value) || 0,
                                            }))
                                        }
                                        className="w-full rounded-xl border border-gray-600 bg-black/60 px-4 py-3 font-mono text-white outline-none transition-all focus:border-wow-gold disabled:opacity-50"
                                    />
                                    {!syncing && (
                                        <button
                                            onClick={() => handleResetProgress(activeSyncType)}
                                            className="absolute right-3 top-1/2 -translate-y-1/2 text-[10px] font-bold uppercase text-red-400 hover:text-red-300"
                                        >
                                            Reset
                                        </button>
                                    )}
                                </div>
                                <p className="mt-2 px-1 text-[10px] text-gray-500">
                                    The sync will process all {activeSyncType}s with Entry ID ≥ this
                                    value.
                                </p>
                            </label>
                        </div>

                        <div className="flex items-start gap-4 rounded-xl border border-blue-500/20 bg-blue-900/10 p-5">
                            <span className="mt-1 text-2xl">ℹ️</span>
                            <div>
                                <div className="mb-1 text-sm font-bold text-blue-200">
                                    Resumable Engine
                                </div>
                                <div className="text-xs leading-relaxed text-blue-100/60">
                                    We remember the last successful ID for each type. You can stop
                                    it anytime and resume from where you left off.
                                </div>
                            </div>
                        </div>
                    </div>

                    {/* Action Buttons */}
                    <div className="space-y-4">
                        {syncing ? (
                            <button
                                onClick={handleStopSync}
                                className="flex w-full animate-pulse items-center justify-center gap-3 rounded-xl border border-red-400/30 bg-red-600 py-4 font-bold text-white shadow-lg transition-all hover:bg-red-500"
                            >
                                <span className="text-xl">⏹</span> STOP SYNCING
                            </button>
                        ) : (
                            <button
                                onClick={handleStartSync}
                                className="flex w-full transform items-center justify-center gap-3 rounded-xl bg-gradient-to-r from-wow-gold to-yellow-500 py-4 font-bold text-gray-900 shadow-[0_0_20px_rgba(198,155,0,0.3)] transition-all hover:scale-[1.01] hover:from-yellow-400 hover:to-wow-gold active:scale-[0.99]"
                            >
                                <span className="text-xl">▶</span> START{' '}
                                {activeSyncType.toUpperCase()} SYNC
                            </button>
                        )}
                    </div>

                    {/* Progress Section */}
                    {(syncing || syncProgress) && (
                        <div className="mt-8 rounded-2xl border border-white/5 bg-black/40 p-6">
                            <div className="mb-4 flex items-end justify-between">
                                <div>
                                    <div className="text-lg font-bold text-wow-gold">
                                        {syncing
                                            ? `Processing ${syncProgress?.type?.toUpperCase() || ''}...`
                                            : 'Paused'}
                                    </div>
                                    <div className="font-mono text-xs text-gray-400">
                                        {syncProgress?.name || 'Waiting...'}
                                    </div>
                                </div>
                                <div className="text-right">
                                    <div className="font-mono text-2xl text-white">
                                        {syncProgress
                                            ? (
                                                  (syncProgress.current / syncProgress.total) *
                                                  100
                                              ).toFixed(1)
                                            : '0.0'}
                                        %
                                    </div>
                                    <div className="text-[10px] font-bold uppercase text-gray-500">
                                        {syncProgress?.current || 0} / {syncProgress?.total || 0}
                                    </div>
                                </div>
                            </div>

                            <div className="mb-6 h-3 w-full overflow-hidden rounded-full bg-gray-900">
                                <div
                                    className="h-full rounded-full bg-wow-gold shadow-[0_0_10px_rgba(198,155,0,0.5)] transition-all duration-500"
                                    style={{
                                        width: `${syncProgress ? (syncProgress.current / syncProgress.total) * 100 : 0}%`,
                                    }}
                                />
                            </div>

                            {/* Log */}
                            <div className="space-y-1.5 border-t border-white/5 pt-4">
                                {syncLog.map((log, idx) => (
                                    <div
                                        key={`${log.id}-${idx}`}
                                        className={`flex items-center gap-3 rounded-lg px-3 py-1.5 font-mono text-xs transition-all ${
                                            idx === 0
                                                ? 'border border-wow-gold/20 bg-wow-gold/10 text-white'
                                                : 'text-gray-500 opacity-60'
                                        }`}
                                    >
                                        <span
                                            className={`h-1.5 w-1.5 rounded-full ${idx === 0 ? 'animate-pulse bg-wow-gold' : 'bg-gray-700'}`}
                                        />
                                        <span className="w-16 opacity-50">#{log.id}</span>
                                        <span className="truncate">{log.name}</span>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}

                    {/* Sync Result Toast */}
                    {syncResult && (
                        <div
                            className={`animate-slideIn mt-6 flex items-start gap-4 rounded-xl border p-4 ${
                                syncResult.error
                                    ? 'border-red-500/30 bg-red-900/20 text-red-200'
                                    : 'border-green-500/30 bg-green-900/20 text-green-200'
                            }`}
                        >
                            <span className="text-xl">{syncResult.error ? '❌' : '✅'}</span>
                            <div>
                                <div className="mb-1 font-bold">
                                    {syncResult.error ? 'Error' : 'Update Status'}
                                </div>
                                <div className="text-sm opacity-80">
                                    {syncResult.error || syncResult.message}
                                </div>
                            </div>
                        </div>
                    )}
                </div>

                {/* Info Cards */}
                <div className="mt-8 grid grid-cols-1 gap-6 md:grid-cols-2">
                    <div className="rounded-2xl border border-gray-700/50 bg-gray-800/30 p-6">
                        <h3 className="mb-3 flex items-center gap-2 font-bold text-wow-gold">
                            🛡️ Safety First
                        </h3>
                        <p className="text-xs leading-relaxed text-gray-400">
                            The sync process is designed to be non-destructive. It updates existing
                            records and adds missing ones while preserving custom fields like
                            'buy_price' if already set manually.
                        </p>
                    </div>
                    <div className="rounded-2xl border border-gray-700/50 bg-gray-800/30 p-6">
                        <h3 className="mb-3 flex items-center gap-2 font-bold text-wow-gold">
                            🚀 Optimization
                        </h3>
                        <p className="text-xs leading-relaxed text-gray-400">
                            NPC, item, quest, and object sync use a multi-threaded worker pool (10
                            workers) to speed up downloads. Spell descriptions are resolved locally
                            from client (DBC) data as part of the Client Data import — no separate
                            sync needed.
                        </p>
                    </div>
                </div>
            </div>
        </PageLayout>
    )
}

export default SyncPage
