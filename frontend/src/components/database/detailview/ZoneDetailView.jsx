import React, { useState } from "react";
import { useZoneDetail } from "../../../hooks/queries/zones";
import { useZoneMap } from "../../../services/useImage";
import {
  DetailPageLayout,
  DetailLoading,
  DetailError,
} from "../../ui";

const ZONE_COLOR = "#4ADE80";

// Plotting every spawn point gets heavy in dense zones; cap the markers we draw.
const MAX_MARKERS = 800;

// Map "services" filters, à la Wowhead. NPC services are creature npc_flags
// bits; books/mailboxes are game-object types. Selecting one narrows the map
// markers and the matching list to just that service.
const SERVICES = [
  { id: "questgiver", label: "Quest Givers", kind: "npc", bit: 0x2 },
  { id: "vendor", label: "Vendors", kind: "npc", bit: 0x80 },
  { id: "trainer", label: "Trainers", kind: "npc", bit: 0x10 | 0x20 | 0x40 },
  { id: "repair", label: "Repairers", kind: "npc", bit: 0x1000 },
  { id: "flightmaster", label: "Flight Masters", kind: "npc", bit: 0x2000 },
  { id: "spirithealer", label: "Spirit Healers", kind: "npc", bit: 0x4000 },
  { id: "innkeeper", label: "Innkeepers", kind: "npc", bit: 0x10000 },
  { id: "banker", label: "Bankers", kind: "npc", bit: 0x20000 },
  { id: "battlemaster", label: "Battlemasters", kind: "npc", bit: 0x100000 },
  { id: "auctioneer", label: "Auctioneers", kind: "npc", bit: 0x200000 },
  { id: "stablemaster", label: "Stable Masters", kind: "npc", bit: 0x400000 },
  { id: "books", label: "Books", kind: "obj", type: 9 },
  { id: "mailbox", label: "Mailboxes", kind: "obj", type: 19 },
];

const npcMatchesService = (n, svc) => svc.kind === "npc" && (n.npcFlags & svc.bit) !== 0;
const objMatchesService = (o, svc) => svc.kind === "obj" && o.type === svc.type;

const ZoneDetailView = ({ entry, onBack, onNavigate }) => {
  const [activeTab, setActiveTab] = useState("npcs");
  const [showMapModal, setShowMapModal] = useState(false);
  const [service, setService] = useState(null); // active service filter id

  const { data: detail, isLoading: loading } = useZoneDetail(entry);

  // Reset the service filter when the zone changes (render-time, no effect).
  const [zoneKey, setZoneKey] = useState(entry);
  if (entry !== zoneKey) {
    setZoneKey(entry);
    setService(null);
  }

  const mapImage = useZoneMap(detail?.mapName);

  if (loading) return <DetailLoading />;
  if (!detail) return <DetailError message="Zone not found" onBack={onBack} />;

  const allNpcs = detail.npcs || [];
  const quests = detail.quests || [];
  const allObjects = detail.objects || [];

  const activeSvc = SERVICES.find((s) => s.id === service) || null;

  // Lists are narrowed to the active service (if any).
  const npcs =
    activeSvc?.kind === "npc" ? allNpcs.filter((n) => npcMatchesService(n, activeSvc)) : allNpcs;
  const objects =
    activeSvc?.kind === "obj" ? allObjects.filter((o) => objMatchesService(o, activeSvc)) : allObjects;

  // Only offer service chips that actually match something in this zone.
  const services = SERVICES.map((s) => ({
    ...s,
    count:
      s.kind === "npc"
        ? allNpcs.filter((n) => npcMatchesService(n, s)).length
        : allObjects.filter((o) => objMatchesService(o, s)).length,
  })).filter((s) => s.count > 0);

  const levelLabel =
    detail.maxLevel > 0
      ? detail.minLevel && detail.minLevel !== detail.maxLevel
        ? `${detail.minLevel} – ${detail.maxLevel}`
        : `${detail.maxLevel}`
      : "—";

  const tabs = [
    { id: "npcs", label: `NPCs (${npcs.length})` },
    { id: "quests", label: `Quests (${quests.length})` },
    { id: "objects", label: `Objects (${objects.length})` },
  ];

  // Map markers: when a service is active, show only its matching spawns;
  // otherwise follow the active tab. Object markers are cyan, creatures emerald.
  let showingObjects;
  let markerSource;
  if (activeSvc) {
    showingObjects = activeSvc.kind === "obj";
    const ok = new Set((showingObjects ? objects : npcs).map((e) => e.entry));
    markerSource = (showingObjects ? detail.objectSpawns : detail.spawns)?.filter((s) =>
      ok.has(s.entry)
    );
  } else {
    showingObjects = activeTab === "objects";
    markerSource = showingObjects ? detail.objectSpawns : detail.spawns;
  }
  const spawns = (markerSource || []).slice(0, MAX_MARKERS);
  const markerClass = showingObjects
    ? "bg-cyan-400/80 border-cyan-200"
    : "bg-emerald-500/80 border-emerald-300";

  const selectService = (s) => {
    if (service === s.id) {
      setService(null);
      return;
    }
    setService(s.id);
    setActiveTab(s.kind === "obj" ? "objects" : "npcs");
  };

  const renderMarkers = (size) =>
    spawns.map((s, idx) => (
      <div
        key={idx}
        className={`absolute rounded-full border shadow ${markerClass}`}
        style={{
          width: size,
          height: size,
          left: `${s.x}%`,
          top: `${s.y}%`,
          marginLeft: -size / 2,
          marginTop: -size / 2,
        }}
      />
    ));

  return (
    <>
      <DetailPageLayout onBack={onBack}>
        {/* Header */}
        <div className="mb-6">
          <h1 className="text-2xl font-bold" style={{ color: ZONE_COLOR }}>
            {detail.name}
          </h1>
          {detail.groupName && (
            <div className="text-sm text-gray-400 mt-1">{detail.groupName}</div>
          )}
        </div>

        {/* Services filter — narrows the map markers and the matching list */}
        {services.length > 0 && (
          <div className="flex flex-wrap gap-1.5 mb-5">
            <button
              onClick={() => setService(null)}
              className={`px-2.5 py-1 rounded text-xs border transition-colors ${
                !service
                  ? "border-wow-gold/60 bg-wow-gold/15 text-wow-gold"
                  : "border-gray-600/40 bg-white/[0.02] text-gray-300 hover:bg-white/5"
              }`}
            >
              All
            </button>
            {services.map((s) => (
              <button
                key={s.id}
                onClick={() => selectService(s)}
                className={`px-2.5 py-1 rounded text-xs border transition-colors ${
                  service === s.id
                    ? "border-wow-gold/60 bg-wow-gold/15 text-wow-gold"
                    : "border-gray-600/40 bg-white/[0.02] text-gray-300 hover:bg-white/5"
                }`}
              >
                {s.label} <span className="text-gray-500">({s.count})</span>
              </button>
            ))}
          </div>
        )}

        <div className="flex flex-col lg:flex-row gap-8">
          {/* Left: Map */}
          <div className="w-full lg:w-[488px] flex-shrink-0">
            <div className="flex justify-between items-baseline border-b border-white/10 pb-1 mb-2">
              <h3 className="text-wow-gold font-bold uppercase text-sm">Map</h3>
              {spawns.length > 0 && (
                <span className="text-xs text-gray-400 font-mono">
                  {spawns.length}
                  {(markerSource?.length || 0) > spawns.length ? "+" : ""}{" "}
                  {showingObjects ? "object" : "spawn"} points
                </span>
              )}
            </div>

            <div
              className="relative w-full aspect-[488/325] bg-cover bg-center rounded border border-white/20 shadow-lg cursor-pointer group overflow-hidden bg-black"
              style={{
                backgroundImage: mapImage.src ? `url(${mapImage.src})` : "none",
              }}
              onClick={() => mapImage.src && setShowMapModal(true)}
            >
              {!mapImage.src && !mapImage.loading && (
                <div className="flex items-center justify-center h-full text-gray-500 text-sm">
                  No Map Data
                </div>
              )}
              {mapImage.loading && (
                <div className="flex items-center justify-center h-full text-gray-500 text-sm animate-pulse">
                  Loading Map...
                </div>
              )}

              {mapImage.src && renderMarkers(8)}

              <div className="absolute top-2 right-2 w-6 h-6 bg-black/50 rounded flex items-center justify-center text-white/80 opacity-0 group-hover:opacity-100 transition-opacity">
                ⤢
              </div>
            </div>
          </div>

          {/* Right: Quick Facts */}
          <div className="flex-1 min-w-0">
            <table className="infobox-table text-sm w-full">
              <thead>
                <tr>
                  <th
                    colSpan="2"
                    className="text-left border-b border-white/10 pb-1 mb-2 text-wow-gold font-bold uppercase text-sm"
                  >
                    Quick Facts
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <th>Region:</th>
                  <td>{detail.groupName || "—"}</td>
                </tr>
                <tr>
                  <th>Creature Levels:</th>
                  <td>{levelLabel}</td>
                </tr>
                <tr>
                  <th>NPCs:</th>
                  <td>{allNpcs.length}</td>
                </tr>
                <tr>
                  <th>Quests:</th>
                  <td>{quests.length}</td>
                </tr>
                <tr>
                  <th>Objects:</th>
                  <td>{allObjects.length}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        {/* Tabs */}
        <div className="border-b border-white/20 mb-4 mt-8 flex gap-1">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`px-4 py-2 text-sm font-bold transition-all relative top-[1px] ${
                activeTab === tab.id
                  ? "tab-btn-active text-white border-b-2 border-wow-gold"
                  : "tab-btn-inactive text-gray-400 hover:text-gray-200"
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>

        <div className="min-h-[200px] animate-fade-in">
          {activeTab === "npcs" && (
            <>
              {npcs.length > 0 ? (
                <div className="bg-bg-sub rounded border border-border-light">
                  {npcs.map((n, i) => (
                    <div
                      key={n.entry}
                      onClick={() => onNavigate("npc", n.entry)}
                      className={`p-3 flex items-center gap-3 hover:bg-white/5 cursor-pointer transition-colors ${
                        i !== npcs.length - 1
                          ? "border-b border-border-light/50"
                          : ""
                      }`}
                    >
                      <span className="text-gray-600 text-[11px] font-mono min-w-[50px]">
                        [{n.entry}]
                      </span>
                      <span className="text-wow-gold hover:text-wow-gold-light text-sm font-medium truncate flex-1">
                        {n.name}
                        {n.subname && (
                          <span className="text-gray-500 ml-1">
                            &lt;{n.subname}&gt;
                          </span>
                        )}
                      </span>
                      <span className="text-gray-500 text-xs whitespace-nowrap">
                        {n.levelMin === n.levelMax
                          ? `Lvl ${n.levelMax}`
                          : `Lvl ${n.levelMin}-${n.levelMax}`}
                        {n.rank > 0 && n.rankName ? ` · ${n.rankName}` : ""}
                      </span>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-gray-500 italic">No NPCs recorded in this zone.</div>
              )}
            </>
          )}

          {activeTab === "quests" && (
            <>
              {quests.length > 0 ? (
                <div className="bg-bg-sub rounded border border-border-light">
                  {quests.map((q, i) => (
                    <div
                      key={q.entry}
                      onClick={() => onNavigate("quest", q.entry)}
                      className={`p-3 flex items-center gap-3 hover:bg-white/5 cursor-pointer transition-colors ${
                        i !== quests.length - 1
                          ? "border-b border-border-light/50"
                          : ""
                      }`}
                    >
                      <span className="text-gray-600 text-[11px] font-mono min-w-[50px]">
                        [{q.entry}]
                      </span>
                      <span className="text-wow-gold hover:text-wow-gold-light text-sm font-medium truncate flex-1">
                        {q.title}
                      </span>
                      {q.questLevel > 0 && (
                        <span className="text-gray-500 text-xs whitespace-nowrap">
                          Lvl {q.questLevel}
                        </span>
                      )}
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-gray-500 italic">No quests in this zone.</div>
              )}
            </>
          )}

          {activeTab === "objects" && (
            <>
              {objects.length > 0 ? (
                <div className="bg-bg-sub rounded border border-border-light">
                  {objects.map((o, i) => (
                    <div
                      key={o.entry}
                      onClick={() => onNavigate("object", o.entry)}
                      className={`p-3 flex items-center gap-3 hover:bg-white/5 cursor-pointer transition-colors ${
                        i !== objects.length - 1
                          ? "border-b border-border-light/50"
                          : ""
                      }`}
                    >
                      <span className="text-gray-600 text-[11px] font-mono min-w-[50px]">
                        [{o.entry}]
                      </span>
                      <span className="text-sm font-medium truncate flex-1" style={{ color: "#4ADE80" }}>
                        {o.name}
                      </span>
                      {o.typeName && (
                        <span className="text-gray-500 text-xs whitespace-nowrap">
                          {o.typeName}
                        </span>
                      )}
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-gray-500 italic">No objects recorded in this zone.</div>
              )}
            </>
          )}
        </div>
      </DetailPageLayout>

      {/* Map Zoom Modal */}
      {showMapModal && mapImage.src && (
        <div
          className="fixed inset-0 bg-black/90 z-50 flex items-center justify-center p-4 cursor-pointer animate-fade-in"
          onClick={() => setShowMapModal(false)}
        >
          <div className="relative max-w-[90vw] max-h-[90vh]" onClick={(e) => e.stopPropagation()}>
            <div className="relative inline-block">
              <img
                src={mapImage.src}
                alt={detail.name}
                className="max-w-full max-h-[85vh] object-contain rounded-lg shadow-2xl"
              />
              {renderMarkers(10)}
            </div>
            <div className="absolute bottom-4 left-1/2 -translate-x-1/2 bg-black/80 px-4 py-2 rounded-lg text-white font-bold">
              {detail.name}
            </div>
            <button
              className="absolute top-2 right-2 w-8 h-8 bg-red-600 hover:bg-red-500 rounded-full text-white font-bold text-lg flex items-center justify-center transition-colors"
              onClick={(e) => {
                e.stopPropagation();
                setShowMapModal(false);
              }}
            >
              ×
            </button>
          </div>
        </div>
      )}
    </>
  );
};

export default ZoneDetailView;
