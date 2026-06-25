import React, { useState, useEffect } from "react";
import {
  GetSyncStats,
  FullSyncNpcs,
  FullSyncItems,
  FullSyncSpells,
  FullSyncQuests,
  StopSync,
} from "../../../wailsjs/go/main/App";
import { EventsOn, EventsOff } from "../../../wailsjs/runtime/runtime";
import { PageLayout } from "../../components/ui";

const SYNC_TYPES = [
  { id: "npc", name: "NPCs", icon: "👤" },
  { id: "item", name: "Items", icon: "⚔️" },
  { id: "spell", name: "Spells", icon: "✨" },
  { id: "quest", name: "Quests", icon: "📜" },
];

function SettingsPage() {
  // Global stats
  const [syncStats, setSyncStats] = useState(null);
  
  // Selection
  const [activeSyncType, setActiveSyncType] = useState(() => {
    return localStorage.getItem('lastActiveSyncType') || "npc";
  });

  // Start IDs (linked to the input field)
  const [startIds, setStartIds] = useState({
      npc: parseInt(localStorage.getItem('lastSyncedNpcId') || "0", 10),
      item: parseInt(localStorage.getItem('lastSyncedItemId') || "0", 10),
      spell: parseInt(localStorage.getItem('lastSyncedSpellId') || "0", 10),
      quest: parseInt(localStorage.getItem('lastSyncedQuestId') || "0", 10),
      model: parseInt(localStorage.getItem('lastSyncedModelId') || "0", 10),
  });

  // Sync state
  const [syncing, setSyncing] = useState(false);
  const [syncResult, setSyncResult] = useState(null);
  const [syncProgress, setSyncProgress] = useState(null);
  const [syncLog, setSyncLog] = useState([]);

  useEffect(() => {
    loadSyncStats();
    
    // NPC Progress
    EventsOn("sync:npc_full:progress", (data) => handleProgress("npc", data));
    EventsOn("sync:npc_full:error", (msg) => handleSyncError("npc", msg));
    EventsOn("sync:npc_full:complete", (msg) => handleSyncDone("npc", msg));

    // Item Progress
    EventsOn("sync:progress", (data) => handleProgress("item", data));
    EventsOn("sync:item_full:error", (msg) => handleSyncError("item", msg));
    EventsOn("sync:item_full:complete", (msg) => handleSyncDone("item", msg));
    
    // Spell Progress
    EventsOn("sync:spells:progress", (data) => handleProgress("spell", data));
    EventsOn("sync:spells_full:complete", (msg) => handleSyncDone("spell", msg));

    // Quest Progress
    EventsOn("sync:quests:progress", (data) => handleProgress("quest", data));
    EventsOn("sync:quests_full:complete", (msg) => handleSyncDone("quest", msg));

    return () => {
        EventsOff("sync:npc_full:progress");
        EventsOff("sync:npc_full:error");
        EventsOff("sync:npc_full:complete");
        EventsOff("sync:progress");
        EventsOff("sync:item_full:error");
        EventsOff("sync:item_full:complete");
        EventsOff("sync:spells:progress");
        EventsOff("sync:spells_full:complete");
        EventsOff("sync:quests:progress");
        EventsOff("sync:quests_full:complete");
    };
  }, []);

  const handleProgress = (type, data) => {
      setSyncProgress({
          type,
          current: data.current,
          total: data.total,
          id: data.itemId || data.id,
          name: data.itemName || `${type.toUpperCase()} ID ${data.itemId || data.id}`
      });

      // Update startIds for resume next time
      const id = data.itemId || data.id;
      setStartIds(prev => ({ ...prev, [type]: id }));
      
      // Persist to localStorage
      const storageKey = `lastSynced${type.charAt(0).toUpperCase() + type.slice(1)}Id`;
      localStorage.setItem(storageKey, id.toString());

      setSyncLog((prev) => {
          if (prev.length > 0 && prev[0].id === id) return prev;
          return [{ id, name: data.itemName || `${type.toUpperCase()} ID ${id}` }, ...prev].slice(0, 5);
      });
  };

  const handleSyncError = (type, msg) => {
      setSyncing(false);
      setSyncResult({ type, error: msg });
      loadSyncStats();
  };

  const handleSyncDone = (type, msg) => {
      setSyncing(false);
      setSyncResult({ type, message: msg || "Sync complete!" });
      loadSyncStats();
  };

  const loadSyncStats = async () => {
    try {
      const stats = await GetSyncStats();
      setSyncStats(stats);
    } catch (error) {
      console.error("Failed to load sync stats:", error);
    }
  };

  const handleStartSync = async () => {
    if (syncing) return;

    setSyncing(true);
    setSyncResult(null);
    setSyncLog([]);
    
    const startId = startIds[activeSyncType];
    localStorage.setItem('lastActiveSyncType', activeSyncType);

    try {
      let result;
      switch (activeSyncType) {
          case 'npc':
              await FullSyncNpcs(startId, 100);
              break;
          case 'item':
              await FullSyncItems(100, true, startId);
              break;
          case 'spell':
              await FullSyncSpells(100, true, startId);
              break;
          case 'quest':
              await FullSyncQuests(100, startId);
              break;
      }
    } catch (error) {
      handleSyncError(activeSyncType, error.toString());
    }
  };

  const handleStopSync = async () => {
    try {
       await StopSync();
       setSyncing(false);
       setSyncResult({
         type: activeSyncType,
         message: "Sync stop requested. It will pause after the current item finishes.",
       });
    } catch (e) {
        console.error(e);
    }
  };

  const handleResetProgress = (type) => {
      if (window.confirm(`Reset progress for ${type.toUpperCase()}?`)) {
          const storageKey = `lastSynced${type.charAt(0).toUpperCase() + type.slice(1)}Id`;
          localStorage.removeItem(storageKey);
          setStartIds(prev => ({ ...prev, [type]: 0 }));
      }
  };

  return (
    <PageLayout>
      <div className="flex-1 overflow-y-auto p-8 max-w-4xl mx-auto w-full">
        <h1 className="text-3xl font-bold text-white mb-8">Data Synchronization</h1>

        {/* Global Stats */}
        {syncStats && (
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
                <div className="bg-gray-800/50 border border-gray-700/50 p-4 rounded-xl">
                    <div className="text-[10px] text-gray-500 uppercase font-bold mb-1">NPCs</div>
                    <div className="text-xl font-mono text-wow-gold">{syncStats.creatureCount}</div>
                </div>
                <div className="bg-gray-800/50 border border-gray-700/50 p-4 rounded-xl">
                    <div className="text-[10px] text-gray-500 uppercase font-bold mb-1">Items</div>
                    <div className="text-xl font-mono text-wow-gold">{syncStats.itemCount}</div>
                </div>
                <div className="bg-gray-800/50 border border-gray-700/50 p-4 rounded-xl">
                    <div className="text-[10px] text-gray-500 uppercase font-bold mb-1">Quests</div>
                    <div className="text-xl font-mono text-wow-gold">{syncStats.questCount}</div>
                </div>
                <div className="bg-gray-800/50 border border-gray-700/50 p-4 rounded-xl">
                    <div className="text-[10px] text-gray-500 uppercase font-bold mb-1">Max Item ID</div>
                    <div className="text-xl font-mono text-gray-400">{syncStats.maxItemID}</div>
                </div>
            </div>
        )}

        <div className="bg-gray-800/40 border border-gray-700 rounded-2xl p-8 shadow-2xl backdrop-blur-sm">
          <div className="flex flex-col md:flex-row md:items-center justify-between gap-6 mb-8">
              <div>
                <h2 className="text-2xl font-bold text-white mb-2">Sync Engine</h2>
                <p className="text-gray-400 text-sm">Download and update database from Web & MySQL sources.</p>
              </div>
              
              {/* Type Switcher */}
              <div className="flex bg-black/40 p-1 rounded-xl border border-white/5">
                {SYNC_TYPES.map(type => (
                    <button
                        key={type.id}
                        disabled={syncing}
                        onClick={() => setActiveSyncType(type.id)}
                        className={`px-4 py-2 rounded-lg text-sm font-bold transition-all ${
                            activeSyncType === type.id 
                            ? "bg-wow-gold text-gray-900 shadow-lg" 
                            : "text-gray-400 hover:text-white"
                        } disabled:opacity-50`}
                    >
                        <span className="mr-2">{type.icon}</span>
                        {type.name}
                    </button>
                ))}
              </div>
          </div>

          {/* Sync Configuration */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-8 mb-8">
              <div className="space-y-4">
                  <label className="block">
                      <span className="text-xs font-bold text-gray-500 uppercase mb-2 block">Starting Entry ID</span>
                      <div className="relative group">
                        <input 
                            type="number"
                            disabled={syncing}
                            value={startIds[activeSyncType]}
                            onChange={(e) => setStartIds(prev => ({ ...prev, [activeSyncType]: parseInt(e.target.value) || 0 }))}
                            className="w-full bg-black/60 border border-gray-600 rounded-xl px-4 py-3 text-white font-mono focus:border-wow-gold outline-none transition-all disabled:opacity-50"
                        />
                        {!syncing && (
                             <button 
                                onClick={() => handleResetProgress(activeSyncType)}
                                className="absolute right-3 top-1/2 -translate-y-1/2 text-[10px] text-red-400 hover:text-red-300 font-bold uppercase"
                             >
                                Reset
                             </button>
                        )}
                      </div>
                      <p className="text-[10px] text-gray-500 mt-2 px-1">
                          The sync will process all {activeSyncType}s with Entry ID ≥ this value.
                      </p>
                  </label>
              </div>

              <div className="bg-blue-900/10 border border-blue-500/20 p-5 rounded-xl flex items-start gap-4">
                  <span className="text-2xl mt-1">ℹ️</span>
                  <div>
                      <div className="text-sm font-bold text-blue-200 mb-1">Resumable Engine</div>
                      <div className="text-xs text-blue-100/60 leading-relaxed">
                          We remember the last successful ID for each type. 
                          You can stop it anytime and resume from where you left off.
                      </div>
                  </div>
              </div>
          </div>

          {/* Action Buttons */}
          <div className="space-y-4">
              {syncing ? (
                <button
                   onClick={handleStopSync}
                   className="w-full bg-red-600 hover:bg-red-500 text-white font-bold py-4 rounded-xl shadow-lg border border-red-400/30 transition-all flex items-center justify-center gap-3 animate-pulse"
                >
                   <span className="text-xl">⏹</span> STOP SYNCING
                </button>
              ) : (
                <button
                    onClick={handleStartSync}
                    className="w-full bg-gradient-to-r from-wow-gold to-yellow-500 hover:from-yellow-400 hover:to-wow-gold text-gray-900 font-bold py-4 rounded-xl shadow-[0_0_20px_rgba(198,155,0,0.3)] transition-all flex items-center justify-center gap-3 transform hover:scale-[1.01] active:scale-[0.99]"
                >
                    <span className="text-xl">▶</span> START {activeSyncType.toUpperCase()} SYNC
                </button>
              )}
          </div>

          {/* Progress Section */}
          {(syncing || syncProgress) && (
              <div className="mt-8 bg-black/40 border border-white/5 rounded-2xl p-6">
                   <div className="flex justify-between items-end mb-4">
                      <div>
                          <div className="text-wow-gold font-bold text-lg">
                              {syncing ? `Processing ${syncProgress?.type?.toUpperCase() || ''}...` : "Paused"}
                          </div>
                          <div className="text-xs text-gray-400 font-mono">
                              {syncProgress?.name || 'Waiting...'}
                          </div>
                      </div>
                      <div className="text-right">
                          <div className="text-2xl font-mono text-white">
                              {syncProgress ? ((syncProgress.current / syncProgress.total) * 100).toFixed(1) : "0.0"}%
                          </div>
                          <div className="text-[10px] text-gray-500 font-bold uppercase">
                              {syncProgress?.current || 0} / {syncProgress?.total || 0}
                          </div>
                      </div>
                  </div>

                  <div className="w-full bg-gray-900 rounded-full h-3 mb-6 overflow-hidden">
                      <div 
                          className="bg-wow-gold h-full rounded-full transition-all duration-500 shadow-[0_0_10px_rgba(198,155,0,0.5)]"
                          style={{ width: `${syncProgress ? (syncProgress.current / syncProgress.total) * 100 : 0}%` }}
                      />
                  </div>

                  {/* Log */}
                  <div className="space-y-1.5 border-t border-white/5 pt-4">
                      {syncLog.map((log, idx) => (
                          <div 
                            key={`${log.id}-${idx}`} 
                            className={`flex items-center gap-3 rounded-lg px-3 py-1.5 text-xs font-mono transition-all ${
                                idx === 0 ? 'bg-wow-gold/10 text-white border border-wow-gold/20' : 'text-gray-500 opacity-60'
                            }`}
                          >
                            <span className={`w-1.5 h-1.5 rounded-full ${idx === 0 ? 'bg-wow-gold animate-pulse' : 'bg-gray-700'}`} />
                            <span className="w-16 opacity-50">#{log.id}</span>
                            <span className="truncate">{log.name}</span>
                          </div>
                      ))}
                  </div>
              </div>
          )}

          {/* Sync Result Toast */}
          {syncResult && (
              <div className={`mt-6 p-4 rounded-xl border flex items-start gap-4 animate-slideIn ${
                  syncResult.error ? 'bg-red-900/20 border-red-500/30 text-red-200' : 'bg-green-900/20 border-green-500/30 text-green-200'
              }`}>
                  <span className="text-xl">{syncResult.error ? "❌" : "✅"}</span>
                  <div>
                      <div className="font-bold mb-1">{syncResult.error ? "Error" : "Update Status"}</div>
                      <div className="text-sm opacity-80">{syncResult.error || syncResult.message}</div>
                  </div>
              </div>
          )}
        </div>

        {/* Info Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mt-8">
            <div className="bg-gray-800/30 border border-gray-700/50 p-6 rounded-2xl">
                <h3 className="text-wow-gold font-bold mb-3 flex items-center gap-2">
                    🛡️ Safety First
                </h3>
                <p className="text-xs text-gray-400 leading-relaxed">
                    The sync process is designed to be non-destructive. 
                    It updates existing records and adds missing ones while preserving custom fields like 'buy_price' if already set manually.
                </p>
            </div>
            <div className="bg-gray-800/30 border border-gray-700/50 p-6 rounded-2xl">
                <h3 className="text-wow-gold font-bold mb-3 flex items-center gap-2">
                    🚀 Optimization
                </h3>
                <p className="text-xs text-gray-400 leading-relaxed">
                    All synchronization (NPCs, items, spells, quests) uses a multi-threaded worker pool (10 workers) to significantly speed up the download process.
                </p>
            </div>
        </div>
      </div>
    </PageLayout>
  );
}

export default SettingsPage;
