import React, { useState, useEffect, useCallback } from "react";
import { EventsOn, EventsOff } from "../../../wailsjs/runtime/runtime";
import { PageLayout } from "../../components/ui";
import { DEFAULT_WOW_BASE } from "../../utils/constants";

const DEFAULT_BASE = DEFAULT_WOW_BASE;

// Each importer maps to an App binding that takes the client base folder.
const IMPORTS = [
  {
    id: "client",
    name: "Client Data (icons, maps, DBC)",
    fn: "RunClientImport",
    sub: "Data\\*.MPQ (or loose DBFilesClient\\ + BlizzardInterfaceArt\\)",
    desc:
      "One pass over your WoW client: decode icons → data/icons, build fully-revealed zone maps → data/maps, and regenerate reference data (zones, skills, quest sorts, factions, item sets, spell text) into the database. Reads straight from the client's MPQ archives in memory when present (nothing is written back).",
  },
  {
    id: "cache",
    name: "WDB Cache",
    fn: "RunCacheImport",
    sub: "WDB\\*.wdb",
    desc:
      "Patch item / quest / creature / gameobject data from your client's WDB caches — everything you've queried in-game. Overlays the freshest server values; existing data is never wiped.",
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
  const [status, setStatus] = useState(null);

  // NPC model rendering (background job with progress events).
  const [modelBusy, setModelBusy] = useState(false);
  const [modelProgress, setModelProgress] = useState(null);
  const [modelMsg, setModelMsg] = useState(null);

  const refreshStatus = useCallback(async () => {
    const app = window?.go?.main?.App;
    if (!app?.GetDataStatus) return;
    try {
      setStatus(await app.GetDataStatus());
    } catch {
      /* ignore */
    }
  }, []);

  useEffect(() => {
    refreshStatus();
  }, [refreshStatus]);

  // Subscribe to model-render progress events.
  useEffect(() => {
    EventsOn("sync:models:progress", (d) =>
      setModelProgress({ current: d.current, total: d.total, name: d.itemName })
    );
    EventsOn("sync:models_full:complete", (msg) => {
      setModelBusy(false);
      setModelMsg({ ok: true, text: msg || "Model render complete" });
      refreshStatus();
    });
    EventsOn("sync:models_full:error", (msg) => {
      setModelBusy(false);
      setModelMsg({ ok: false, text: msg });
    });
    return () => {
      EventsOff("sync:models:progress");
      EventsOff("sync:models_full:complete");
      EventsOff("sync:models_full:error");
    };
  }, [refreshStatus]);

  const runModelRender = () => {
    const app = window?.go?.main?.App;
    if (!app?.RenderNpcModels) {
      setModelMsg({ ok: false, text: "Binding not found (restart dev build)" });
      return;
    }
    setModelBusy(true);
    setModelMsg(null);
    setModelProgress(null);
    app.RenderNpcModels(base, 0, 50);
  };

  const stopModelRender = () => {
    window?.go?.main?.App?.StopSync?.();
    setModelBusy(false);
  };

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
      refreshStatus();
    }
  };

  // Categories that are populated by an import and worth warning about when empty.
  const missing = status
    ? [
        status.icons === 0 && "icons",
        status.maps === 0 && "zone maps",
      ].filter(Boolean)
    : [];

  return (
    <PageLayout>
      <div className="max-w-3xl mx-auto p-6 space-y-6 overflow-y-auto h-full">
        <div>
          <h2 className="text-xl text-wow-gold font-bold mb-1">Import</h2>
          <p className="text-gray-400 text-sm">
            Refresh InkLab's data from your local WoW client. Nothing is uploaded — each
            import reads the files under the folder below.
          </p>
        </div>

        {missing.length > 0 && (
          <div className="rounded-xl border border-amber-500/40 bg-amber-500/10 p-4">
            <div className="text-amber-300 font-semibold text-sm">
              ⚠️ No {missing.join(" or ")} found
            </div>
            <p className="text-amber-200/80 text-sm mt-1">
              InkLab ships without bundled {missing.join(" / ")} — they're built from
              your local WoW client. Set the client folder below and run the matching
              import, or items will show a placeholder icon and NPCs won't show a zone map.
            </p>
          </div>
        )}

        {status && (
          <div className="flex flex-wrap gap-2 text-[11px]">
            {[
              ["Icons", status.icons],
              ["Zone maps", status.maps],
              ["NPC images", status.npcImages],
            ].map(([label, n]) => (
              <span
                key={label}
                className={`px-2 py-1 rounded border font-mono ${
                  n > 0
                    ? "border-green-600/40 bg-green-600/10 text-green-300"
                    : "border-gray-600/40 bg-gray-600/10 text-gray-400"
                }`}
              >
                {label}: {n.toLocaleString()}
              </span>
            ))}
          </div>
        )}

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

        {/* NPC Model Renders — renders creature models from the client MPQs.
            Background job with progress; falls back to octowow per-display for
            humanoid character models. */}
        <div className="bg-gray-800/50 border border-gray-700/50 rounded-xl p-4">
          <div className="flex items-start justify-between gap-4">
            <div className="min-w-0">
              <h3 className="text-white font-semibold">NPC Model Renders</h3>
              <p className="text-gray-400 text-sm mt-1">
                Render creature models straight from your client into
                data/npc_images, keyed by display id. Humanoid character models
                (and any render failure) fall back to octowow's pre-rendered
                image. Runs in the background; existing images are kept.
              </p>
              <p className="text-[11px] text-gray-600 font-mono mt-1">
                Data\*.MPQ → Creature\…\*.m2 + skins
              </p>
            </div>
            {modelBusy ? (
              <button
                onClick={stopModelRender}
                className="shrink-0 bg-red-600 hover:bg-red-500 text-white font-bold px-5 py-2 rounded transition-colors"
              >
                Stop
              </button>
            ) : (
              <button
                onClick={runModelRender}
                disabled={!!running}
                className="shrink-0 bg-wow-gold/90 hover:bg-wow-gold text-black font-bold px-5 py-2 rounded transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              >
                Render
              </button>
            )}
          </div>

          {modelProgress && (
            <div className="mt-3">
              <div className="flex justify-between text-[11px] text-gray-400 font-mono mb-1">
                <span>{modelProgress.name || "Rendering…"}</span>
                <span>
                  {modelProgress.current} / {modelProgress.total}
                </span>
              </div>
              <div className="w-full bg-black/40 rounded-full h-2 overflow-hidden">
                <div
                  className="bg-wow-gold h-full rounded-full transition-all"
                  style={{
                    width: `${modelProgress.total ? (modelProgress.current / modelProgress.total) * 100 : 0}%`,
                  }}
                />
              </div>
            </div>
          )}
          {modelMsg && (
            <div
              className={`mt-3 rounded border p-3 text-sm ${
                modelMsg.ok
                  ? "border-green-500/30 bg-green-500/5 text-green-400"
                  : "border-red-500/30 bg-red-500/5 text-red-400"
              }`}
            >
              {modelMsg.text}
            </div>
          )}
        </div>

        {/* What's New — diff of the live DB vs the baseline. Placed last: it's
            noise for a brand-new user who hasn't imported anything yet. */}
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
      </div>
    </PageLayout>
  );
}

export default ToolsPage;
