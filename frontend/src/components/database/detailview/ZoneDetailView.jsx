import React, { useState, useEffect } from "react";
import { GetZoneDetail } from "../../../utils/databaseApi";
import { useZoneMap } from "../../../services/useImage";
import {
  DetailPageLayout,
  DetailSection,
  DetailLoading,
  DetailError,
} from "../../ui";

const ZONE_COLOR = "#4ADE80";

// Plotting every spawn point gets heavy in dense zones; cap the markers we draw.
const MAX_MARKERS = 800;

const ZoneDetailView = ({ entry, onBack, onNavigate }) => {
  const [activeTab, setActiveTab] = useState("npcs");
  const [detail, setDetail] = useState(null);
  const [loading, setLoading] = useState(true);
  const [showMapModal, setShowMapModal] = useState(false);

  const mapImage = useZoneMap(detail?.mapName);

  useEffect(() => {
    setLoading(true);
    GetZoneDetail(entry).then((res) => {
      setDetail(res);
      setLoading(false);
    });
  }, [entry]);

  if (loading) return <DetailLoading />;
  if (!detail) return <DetailError message="Zone not found" onBack={onBack} />;

  const npcs = detail.npcs || [];
  const quests = detail.quests || [];
  const spawns = (detail.spawns || []).slice(0, MAX_MARKERS);

  const levelLabel =
    detail.maxLevel > 0
      ? detail.minLevel && detail.minLevel !== detail.maxLevel
        ? `${detail.minLevel} – ${detail.maxLevel}`
        : `${detail.maxLevel}`
      : "—";

  const tabs = [
    { id: "npcs", label: `NPCs (${npcs.length})` },
    { id: "quests", label: `Quests (${quests.length})` },
  ];

  const renderMarkers = (size) =>
    spawns.map((s, idx) => (
      <div
        key={idx}
        className="absolute rounded-full bg-emerald-500/80 border border-emerald-300 shadow"
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

        <div className="flex flex-col lg:flex-row gap-8">
          {/* Left: Map */}
          <div className="w-full lg:w-[488px] flex-shrink-0">
            <div className="flex justify-between items-baseline border-b border-white/10 pb-1 mb-2">
              <h3 className="text-wow-gold font-bold uppercase text-sm">Map</h3>
              {spawns.length > 0 && (
                <span className="text-xs text-gray-400 font-mono">
                  {spawns.length}
                  {(detail.spawns?.length || 0) > spawns.length ? "+" : ""} spawn
                  points
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
                  <td>{npcs.length}</td>
                </tr>
                <tr>
                  <th>Quests:</th>
                  <td>{quests.length}</td>
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
            <DetailSection title={`NPCs (${npcs.length})`}>
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
            </DetailSection>
          )}

          {activeTab === "quests" && (
            <DetailSection title={`Quests (${quests.length})`}>
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
            </DetailSection>
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
