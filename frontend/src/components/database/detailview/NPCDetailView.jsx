import React, { useState, useEffect } from "react";
import { GetNpcFullDetails, SyncNpcData, RefreshNpcImages } from "../../../services/api";
import { useNpcModel, useZoneMap, useIcon } from "../../../services/useImage";
import { evictImage } from "../../../services/imageService";
import { getQualityColor, formatMoney } from "../../../utils/wow";
import { DATABASE_BASE_URL } from "../../../utils/constants";
import {
  DetailPageLayout,
  DetailHeader,
  DetailSection,
  DetailGrid,
  LootGrid,
  StatBadge,
  DetailLoading,
  DetailError,
  LootItem,
} from "../../ui";

// AbilityIcon resolves a spell's icon through the local icon service (local
// data/icons first, then CDN, then the questionmark placeholder) — same as the
// rest of the app. The previous remote-only <img> 404'd on octo-custom icons
// and hid itself, which looked like the icon flashing then disappearing.
const AbilityIcon = ({ iconName }) => {
  const icon = useIcon(iconName);
  if (!iconName) return null;
  return (
    <img
      src={icon.src || "/local-icons/inv_misc_questionmark.jpg"}
      alt=""
      className="w-10 h-10 rounded border border-gray-600 bg-black/40"
    />
  );
};

const NPCDetailView = ({ entry, onBack, onNavigate, tooltipHook }) => {
  const [activeTab, setActiveTab] = useState("overview");
  const [detail, setDetail] = useState(null);
  const [loading, setLoading] = useState(true);
  const [showMapModal, setShowMapModal] = useState(false);
  const [imgReload, setImgReload] = useState(0);
  const [refreshingImages, setRefreshingImages] = useState(false);

  // Model renders are produced locally from the client MPQs (per-creature when
  // the NPC has weapons, else the shared display render) — no network fetching.
  // The map is a locally-generated zone map keyed by the NPC's resolved zone name.
  const displayId = detail?.displayId1;
  // Pass the creature entry so a per-creature render (with held weapons) is
  // preferred over the shared display render when one exists locally.
  const modelImage = useNpcModel(displayId, imgReload, entry);
  const mapImage = useZoneMap(detail?.zoneName, imgReload);

  useEffect(() => {
    setLoading(true);
    GetNpcFullDetails(entry).then((res) => {
      setDetail(res);
      setLoading(false);
    });
  }, [entry]);

  const handleSync = () => {
    setLoading(true);
    SyncNpcData(entry).then((res) => {
      setDetail(res);
      setLoading(false);
    });
  };

  // Re-fetch just the model/map images (does NOT replace creature data).
  const handleRefreshImages = () => {
    if (refreshingImages) return;
    setRefreshingImages(true);
    RefreshNpcImages(entry)
      .then((res) => {
        if (res) setDetail(res);
        if (displayId) evictImage("npc_model", `model_${displayId}`);
        evictImage("npc_model", `model_creature_${entry}`);
        setImgReload((n) => n + 1);
      })
      .finally(() => setRefreshingImages(false));
  };

  const renderLootItem = (item) => {
    const handlers = tooltipHook?.getItemHandlers?.(item.itemId) || {};
    return (
      <LootItem
        key={item.itemId}
        item={{
          entry: item.itemId,
          name: item.name,
          quality: item.quality,
          iconPath: "", // Icon paths might be missing in scraping-only mode, but DB should have them if joined
          dropChance: `${item.chance.toFixed(1)}%`,
        }}
        onClick={() => onNavigate("item", item.itemId)}
        showDropChance
        {...handlers}
      />
    );
  };

  if (loading) return <DetailLoading />;
  if (!detail) return <DetailError message="NPC not found" onBack={onBack} />;

  const startsQuests = detail.quests?.filter((q) => q.type === "starts") || [];
  const endsQuests = detail.quests?.filter((q) => q.type === "ends") || [];
  const loot = detail.loot || [];
  const abilities = detail.abilities || [];
  const sells = detail.sells || [];

  const tabs = [
    { id: "overview", label: "Overview" },
    { id: "loot", label: `Loot (${loot.length})` },
    {
      id: "quests",
      label: `Quests (${startsQuests.length + endsQuests.length})`,
    },
    { id: "abilities", label: `Abilities (${abilities.length})` },
    ...(sells.length > 0 ? [{ id: "sells", label: `Sells (${sells.length})` }] : []),
  ];

  return (
    <>
      <DetailPageLayout onBack={onBack}>
        {/* --- Header Section --- */}
        <div className="mb-6 flex justify-between items-start">
          <div>
            <h1
              className={`text-2xl font-bold ${getQualityColor(
                detail.rank >= 1 ? 5 : 1
              )}`}
            >
              {detail.name}
            </h1>
            {detail.subname && (
              <div className="text-sm text-yellow-200 mt-1">
                &lt;{detail.subname}&gt;
              </div>
            )}
          </div>
          <div className="flex gap-2">
            <button
              onClick={handleSync}
              className="px-3 py-1 bg-blue-600 hover:bg-blue-500 text-white text-xs font-bold rounded border border-blue-700 transition-colors flex items-center gap-1"
              title="Re-download data from external sources"
            >
              <span className="text-sm">↻</span> Sync
            </button>
            <a
              href={`${DATABASE_BASE_URL}/?npc=${detail.entry}`}
              target="_blank"
              rel="noreferrer"
              className="px-3 py-1 bg-purple-700 hover:bg-purple-600 text-white text-xs font-bold rounded border border-purple-800 transition-colors"
              title="View on Turtle WoW Database"
            >
              🔗 OctoHead
            </a>
            <a
              href={`https://www.wowhead.com/classic/npc=${detail.entry}`}
              target="_blank"
              rel="noreferrer"
              className="px-3 py-1 bg-red-800 hover:bg-red-700 text-white text-xs font-bold rounded border border-red-900 transition-colors"
            >
              Wowhead
            </a>
          </div>
        </div>

        <div className="flex flex-col lg:flex-row gap-8">
          {/* --- Left Column: Visuals (Model Only) --- */}
          <div className="w-full lg:w-64 flex-shrink-0 space-y-4">
            {/* Model Image (if available) - Centered or Top aligned */}
            {modelImage.loading ? (
              <div className="aspect-[3/4] border border-white/10 rounded bg-black/40 flex items-center justify-center text-gray-500 text-xs animate-pulse">
                Loading...
              </div>
            ) : modelImage.src ? (
              <div className="border border-white/20 rounded bg-black overflow-hidden shadow-lg mb-4">
                <img
                  src={modelImage.src}
                  alt={detail.name}
                  className="w-full h-auto object-cover"
                />
              </div>
            ) : (
              <div
                onClick={handleRefreshImages}
                title="Click to render model from client"
                className="aspect-[3/4] border border-white/10 rounded bg-black/40 flex flex-col items-center justify-center text-gray-500 text-xs cursor-pointer hover:bg-black/60 hover:text-gray-300 transition-colors"
              >
                {refreshingImages ? (
                  <span className="animate-pulse">Rendering…</span>
                ) : (
                  <>
                    <span>No Model</span>
                    <span className="mt-1 text-[10px] text-gray-600">click to render</span>
                  </>
                )}
              </div>
            )}
          </div>

          {/* --- Right Column: Data & Tabs --- */}
          <div className="flex-1 min-w-0">
            {/* Top Section: Location & Quick Facts Side-by-Side */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
              {/* Location Box (Updated Style) */}
              <div className="h-fit">
                <div className="flex justify-between items-baseline border-b border-white/10 pb-1 mb-2">
                  <h3 className="text-wow-gold font-bold uppercase text-sm">
                    Location
                  </h3>
                  {detail.spawns && detail.spawns.length > 0 && (
                    <span className="text-xs text-gray-400 font-mono">
                      {detail.spawns[0].zoneName || `Map ${detail.spawns[0].mapId}`}
                      {(detail.spawns[0].x > 0 || detail.spawns[0].y > 0) && (
                        <span className="ml-1">
                          ({detail.spawns[0].x.toFixed(1)}, {detail.spawns[0].y.toFixed(1)})
                        </span>
                      )}
                    </span>
                  )}
                </div>

                {/* User Recommended Map Style with Spawn Markers */}
                <div
                  className="mapper-map relative w-full aspect-[488/325] bg-cover bg-center rounded border border-white/20 shadow-lg cursor-pointer group overflow-hidden bg-black"
                  style={{
                    backgroundImage: mapImage.src
                      ? `url(${mapImage.src})`
                      : "none",
                    maxWidth: "488px",
                    maxHeight: "325px",
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

                  {/* Spawn Point Markers */}
                  {mapImage.src && detail.spawns && detail.spawns.map((spawn, idx) => {
                    // Only show markers for coordinates in valid 0-100 range
                    const hasValidCoords = spawn.x > 0 && spawn.x <= 100 && spawn.y > 0 && spawn.y <= 100;
                    if (!hasValidCoords) return null;
                    
                    return (
                      <div
                        key={idx}
                        className="absolute w-4 h-4 -ml-2 -mt-2 cursor-pointer group/marker z-10"
                        style={{
                          left: `${spawn.x}%`,
                          top: `${spawn.y}%`,
                        }}
                        onClick={(e) => e.stopPropagation()}
                      >
                        {/* Outer pulsing ring */}
                        <div className="absolute inset-0 bg-red-500/50 rounded-full animate-ping" />
                        {/* Inner solid dot */}
                        <div className="absolute inset-0.5 bg-red-600 rounded-full border border-red-400 shadow-lg" />
                        
                        {/* Tooltip on hover */}
                        <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 px-2 py-1 bg-black/90 rounded text-xs text-white whitespace-nowrap opacity-0 group-hover/marker:opacity-100 transition-opacity pointer-events-none border border-white/20 shadow-lg">
                          <div className="font-semibold text-wow-gold">{spawn.zoneName || 'Spawn Point'}</div>
                          <div className="text-gray-300 font-mono">({spawn.x.toFixed(1)}, {spawn.y.toFixed(1)})</div>
                          {/* Arrow */}
                          <div className="absolute top-full left-1/2 -translate-x-1/2 border-4 border-transparent border-t-black/90" />
                        </div>
                      </div>
                    );
                  })}

                  {/* Overlay / Expander */}
                  <div className="absolute inset-0 bg-transparent group-hover:bg-white/5 transition-colors pointer-events-none"></div>
                  <div className="absolute top-2 right-2 w-6 h-6 bg-black/50 rounded flex items-center justify-center text-white/80 opacity-0 group-hover:opacity-100 transition-opacity">
                    ⤢
                  </div>

                  {/* Spawn Count Badge */}
                  {detail.spawns && detail.spawns.length > 1 && (
                    <div className="absolute top-2 left-2 px-2 py-0.5 bg-black/70 rounded text-xs text-gray-300 border border-white/10">
                      {detail.spawns.length} spawns
                    </div>
                  )}

                  {/* Zoom Tip */}
                  <div className="absolute bottom-0 left-0 right-0 bg-black/80 text-center py-1 text-xs text-gray-300 opacity-0 group-hover:opacity-100 transition-opacity">
                    Tip: Click map to zoom
                  </div>
                </div>
              </div>

              {/* Quick Facts / Stats Block */}
              <div>
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
                      <th>Level:</th>
                      <td>
                        {detail.levelMin !== detail.levelMax
                          ? `${detail.levelMin} - ${detail.levelMax}`
                          : detail.levelMax}
                      </td>
                    </tr>
                    <tr>
                      <th>Classification:</th>
                      <td>{detail.rankName || detail.rank}</td>
                    </tr>
                    <tr>
                      <th>React:</th>
                      <td>
                        <span
                          className={
                            detail.faction === 35
                              ? "text-wow-quality-2"
                              : "text-wow-quality-7"
                          }
                        >
                          A
                        </span>{" "}
                        <span
                          className={
                            detail.faction === 35
                              ? "text-wow-quality-2"
                              : "text-wow-quality-7"
                          }
                        >
                          H
                        </span>
                      </td>
                    </tr>
                    <tr>
                      <th>Faction:</th>
                      <td>{detail.faction}</td>
                    </tr>
                    <tr>
                      <th>Health:</th>
                      <td>
                        {detail.healthMin !== detail.healthMax
                          ? `${detail.healthMin} - ${detail.healthMax}`
                          : detail.healthMax}
                      </td>
                    </tr>
                    {(detail.manaMin > 0 || detail.manaMax > 0) && (
                      <tr>
                        <th>Mana:</th>
                        <td>
                          {detail.manaMin !== detail.manaMax
                            ? `${detail.manaMin} - ${detail.manaMax}`
                            : detail.manaMax}
                        </td>
                      </tr>
                    )}
                    {(detail.goldMin > 0 || detail.goldMax > 0) && (
                      <tr>
                        <th>Wealth:</th>
                        <td>
                          {/* Simple money format for now */}
                          {(detail.goldMin / 10000).toFixed(2)}g -{" "}
                          {(detail.goldMax / 10000).toFixed(2)}g
                        </td>
                      </tr>
                    )}
                    {(detail.minDmg > 0 || detail.maxDmg > 0) && (
                      <tr>
                        <th>Damage:</th>
                        <td>
                          {Math.floor(detail.minDmg)} -{" "}
                          {Math.floor(detail.maxDmg)}
                        </td>
                      </tr>
                    )}
                    {detail.armor > 0 && (
                      <tr>
                        <th>Armor:</th>
                        <td>{detail.armor}</td>
                      </tr>
                    )}
                    <tr>
                      <th>Display ID:</th>
                      <td>{detail.displayId1}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>

            {/* Tabs Navigation */}
            <div className="border-b border-white/20 mb-4 flex gap-1">
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

            {/* Tab Content */}
            <div className="min-h-[200px] animate-fade-in">
              {activeTab === "overview" && (
                <div className="text-gray-400 text-sm">
                  <h4 className="text-white font-bold mb-2">
                    Abilities Summary
                  </h4>
                  {detail.abilities?.length > 0 ? (
                    <ul className="list-disc pl-5 space-y-1">
                      {detail.abilities.slice(0, 5).map((spell, i) => (
                        <li key={spell.id || i}>
                          <span className="text-wow-quality-1">
                            {spell.name}
                          </span>
                        </li>
                      ))}
                    </ul>
                  ) : (
                    "No abilities found."
                  )}
                </div>
              )}

              {activeTab === "loot" && (
                <div className="animate-fade-in">
                  {loot.length > 0 ? (
                    <LootGrid>
                      {loot
                        .sort((a, b) => b.chance - a.chance)
                        .map((item) => {
                          // Support both entry and itemId for compatibility
                          const itemId = item.entry || item.itemId;
                          const handlers =
                            tooltipHook?.getItemHandlers?.(itemId) || {};
                          return (
                            <LootItem
                              key={itemId}
                              item={{
                                entry: itemId,
                                name: item.name,
                                quality: item.quality,
                                iconPath: item.iconPath || item.icon || "",
                                dropChance: `${item.chance.toFixed(1)}%`,
                              }}
                              onClick={() => onNavigate("item", itemId)}
                              showDropChance
                              {...handlers}
                            />
                          );
                        })}
                    </LootGrid>
                  ) : (
                    <div className="p-8 text-center text-gray-500 italic">
                      No loot table available.
                    </div>
                  )}
                </div>
              )}

              {activeTab === "sells" && (
                <div className="animate-fade-in">
                  <LootGrid>
                    {sells.map((item) => {
                      const handlers =
                        tooltipHook?.getItemHandlers?.(item.itemId) || {};
                      const m = item.cost > 0 ? formatMoney(item.cost) : null;
                      const price = m
                        ? [m.g && `${m.g}g`, m.s && `${m.s}s`, m.c && `${m.c}c`]
                            .filter(Boolean)
                            .join(" ")
                        : "";
                      return (
                        <LootItem
                          key={item.itemId}
                          item={{
                            entry: item.itemId,
                            name: item.name || `Item ${item.itemId}`,
                            quality: item.quality,
                            iconPath: item.iconPath || "",
                            dropChance: price,
                          }}
                          onClick={() => onNavigate("item", item.itemId)}
                          showDropChance={!!price}
                          {...handlers}
                        />
                      );
                    })}
                  </LootGrid>
                </div>
              )}

              {activeTab === "quests" && (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6 animate-fade-in">
                  <DetailSection
                    title={`Starts Quests (${startsQuests.length})`}
                  >
                    {startsQuests.length > 0 ? (
                      <div className="bg-bg-sub rounded border border-border-light">
                        {startsQuests.map((q, i) => (
                          <div
                            key={q.entry || q.questId}
                            onClick={() =>
                              onNavigate("quest", q.entry || q.questId)
                            }
                            className={`p-3 flex items-center justify-between hover:bg-white/5 cursor-pointer transition-colors ${
                              i !== startsQuests.length - 1
                                ? "border-b border-border-light/50"
                                : ""
                            }`}
                          >
                            <span className="text-wow-gold hover:text-wow-gold-light md:text-sm font-medium truncate">
                              {q.name || q.title}
                            </span>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <div className="text-gray-500 italic">None</div>
                    )}
                  </DetailSection>

                  <DetailSection title={`Ends Quests (${endsQuests.length})`}>
                    {endsQuests.length > 0 ? (
                      <div className="bg-bg-sub rounded border border-border-light">
                        {endsQuests.map((q, i) => (
                          <div
                            key={q.entry || q.questId}
                            onClick={() =>
                              onNavigate("quest", q.entry || q.questId)
                            }
                            className={`p-3 flex items-center justify-between hover:bg-white/5 cursor-pointer transition-colors ${
                              i !== endsQuests.length - 1
                                ? "border-b border-border-light/50"
                                : ""
                            }`}
                          >
                            <span className="text-wow-gold hover:text-wow-gold-light md:text-sm font-medium truncate">
                              {q.name || q.title}
                            </span>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <div className="text-gray-500 italic">None</div>
                    )}
                  </DetailSection>
                </div>
              )}

              {activeTab === "abilities" && (
                <div className="animate-fade-in">
                  {abilities.length > 0 ? (
                    <div className="grid grid-cols-1 gap-4">
                      {abilities.map((spell, idx) => (
                        <div
                          key={spell.id || idx}
                          className="bg-bg-sub p-4 rounded border border-border-light hover:border-border-hover transition-colors"
                        >
                            <div className="flex justify-between items-start mb-2">
                              <div className="flex items-center gap-3">
                                <AbilityIcon iconName={spell.icon} />
                                <h4 className="text-wow-quality-1 font-bold text-lg">
                                  {spell.name}
                                </h4>
                              </div>
                            </div>
                          <p className="text-gray-300 text-sm leading-relaxed pl-[3.25rem]">
                            {spell.description &&
                            spell.description.length > 2 ? (
                              spell.description
                            ) : (
                              <span className="text-gray-600 italic">
                                No description available.
                              </span>
                            )}
                          </p>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <div className="p-8 text-center text-gray-500 italic">
                      No abilities data found.
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        </div>
      </DetailPageLayout>

      {/* Map Zoom Modal */}
      {showMapModal && mapImage.src && (
        <div
          className="fixed inset-0 bg-black/90 z-50 flex items-center justify-center p-4 cursor-pointer animate-fade-in"
          onClick={() => setShowMapModal(false)}
        >
          <div 
            className="relative max-w-[90vw] max-h-[90vh]"
            onClick={(e) => e.stopPropagation()}
          >
            {/* Map Image Container with Markers */}
            <div className="relative">
              <img
                src={mapImage.src}
                alt={detail.zoneName || "Location Map"}
                className="max-w-full max-h-[85vh] object-contain rounded-lg shadow-2xl"
              />
              
              {/* Spawn Point Markers on Modal Map */}
              {detail.spawns && detail.spawns.map((spawn, idx) => {
                const hasValidCoords = spawn.x > 0 && spawn.x <= 100 && spawn.y > 0 && spawn.y <= 100;
                if (!hasValidCoords) return null;
                
                return (
                  <div
                    key={idx}
                    className="absolute w-5 h-5 -ml-2.5 -mt-2.5 pointer-events-none"
                    style={{
                      left: `${spawn.x}%`,
                      top: `${spawn.y}%`,
                    }}
                    title={`${spawn.zoneName || 'Spawn'} (${spawn.x.toFixed(1)}, ${spawn.y.toFixed(1)})`}
                  >
                    {/* Outer pulsing ring */}
                    <div className="absolute inset-0 bg-red-500/50 rounded-full animate-ping" />
                    {/* Inner solid dot */}
                    <div className="absolute inset-1 bg-red-600 rounded-full border-2 border-red-400 shadow-lg" />
                  </div>
                );
              })}
            </div>

            {/* Zone Name Label */}
            {(detail.zoneName || (detail.spawns && detail.spawns.length > 0)) && (
              <div className="absolute bottom-4 left-1/2 -translate-x-1/2 bg-black/80 px-4 py-2 rounded-lg text-white font-bold">
                {detail.zoneName || detail.spawns?.[0]?.zoneName || "Unknown Zone"}
                {detail.x > 0 &&
                  ` (${detail.x.toFixed(1)}, ${detail.y.toFixed(1)})`}
                {detail.spawns && detail.spawns.length > 1 && (
                  <span className="ml-2 text-gray-400 text-sm">
                    +{detail.spawns.length - 1} more
                  </span>
                )}
              </div>
            )}
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

export default NPCDetailView;
