import React, { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { queryClient } from "../../../queryClient";
import { DATABASE_BASE_URL } from "../../../utils/constants";
import { GetObjectDetail } from "../../../../wailsjs/go/main/App";
import { useZoneMap } from "../../../services/useImage";
import {
  DetailPageLayout,
  DetailHeader,
  DetailSection,
  DetailLoading,
  DetailError,
  LootItem,
  LootGrid,
} from "../../ui";

const ObjectDetailView = ({ entry, onBack, onNavigate, tooltipHook }) => {
  const [showMapModal, setShowMapModal] = useState(false);
  const [selectedZone, setSelectedZone] = useState(null);
  const [syncing, setSyncing] = useState(false);

  const { data: detail, isLoading: loading } = useQuery({
    queryKey: ["objectDetail", entry],
    queryFn: () => GetObjectDetail(entry),
    enabled: entry != null,
  });

  // Reset the selected zone when the object changes (render-time, no effect).
  const [objKey, setObjKey] = useState(entry);
  if (entry !== objKey) {
    setObjKey(entry);
    setSelectedZone(null);
  }

  const spawns = detail?.spawns || [];

  // Spawns span many zones (a mailbox is in every Alliance town). Group by zone
  // so each zone's markers plot on its own map — plotting them all on one map
  // would scatter them at nonsense positions.
  const zones = useMemo(() => {
    const m = new Map();
    for (const s of spawns) {
      const name = s.zoneName || `Map ${s.mapId}`;
      if (!m.has(name)) m.set(name, []);
      m.get(name).push(s);
    }
    return [...m.entries()]
      .map(([name, pts]) => ({ name, pts }))
      .sort((a, b) => b.pts.length - a.pts.length);
  }, [spawns]);

  const activeZone =
    (selectedZone && zones.find((z) => z.name === selectedZone)) || zones[0] || null;
  const mapImage = useZoneMap(activeZone?.name);

  const handleSync = () => {
    if (syncing) return;
    const fn = window?.go?.main?.App?.SyncObjectSpawns;
    if (!fn) return;
    setSyncing(true);
    fn(entry)
      .then((res) => {
        if (res) queryClient.setQueryData(["objectDetail", entry], res);
        setSelectedZone(null);
      })
      .finally(() => setSyncing(false));
  };

  if (loading) return <DetailLoading />;
  if (!detail) return <DetailError message="Object not found" onBack={onBack} />;

  const startsQuests = detail.startsQuests || [];
  const endsQuests = detail.endsQuests || [];
  const contains = detail.contains || [];

  const markers = (activeZone?.pts || []).filter(
    (s) => s.x > 0 && s.x <= 100 && s.y > 0 && s.y <= 100
  );
  const renderMarkers = (size) =>
    markers.map((s, idx) => (
      <div
        key={idx}
        className="absolute -ml-2 -mt-2 z-10"
        style={{ left: `${s.x}%`, top: `${s.y}%`, width: size, height: size }}
      >
        <div className="absolute inset-0 bg-red-500/50 rounded-full animate-ping" />
        <div className="absolute inset-0.5 bg-red-600 rounded-full border border-red-400 shadow-lg" />
      </div>
    ));

  return (
    <>
      <DetailPageLayout onBack={onBack}>
        <DetailHeader
          icon={
            <div className="w-full h-full flex items-center justify-center bg-gray-800 text-3xl">
              🗿
            </div>
          }
          iconBorderColor="text-gray-400"
          title={detail.name}
          titleColor="text-white"
          subtitle={`${detail.typeName || 'Object'} • ID: ${detail.entry}`}
          action={
            <div className="flex gap-2">
              <button
                onClick={handleSync}
                disabled={syncing}
                title="Fetch spawn locations from octowow.st"
                className="px-3 py-1.5 text-xs font-bold uppercase rounded transition-colors bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white flex items-center gap-1"
              >
                <span className="text-sm">↻</span> {syncing ? "Syncing…" : "Sync Spawns"}
              </button>
              <a
                href={`${DATABASE_BASE_URL}/?object=${detail.entry}`}
                target="_blank"
                rel="noreferrer"
                className="px-3 py-1.5 text-xs font-bold uppercase rounded transition-colors bg-purple-700 hover:bg-purple-600 text-white"
              >
                🔗 OctoHead
              </a>
            </div>
          }
        />

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
          {/* Quick Facts */}
          <DetailSection title="Quick Facts">
            <table className="infobox-table text-sm w-full">
              <tbody>
                <tr>
                  <th className="text-gray-400 pr-4 py-1">Type:</th>
                  <td className="text-white">{detail.typeName || detail.type}</td>
                </tr>
                <tr>
                  <th className="text-gray-400 pr-4 py-1">Display ID:</th>
                  <td className="text-white">{detail.displayId}</td>
                </tr>
                {detail.faction > 0 && (
                  <tr>
                    <th className="text-gray-400 pr-4 py-1">Faction:</th>
                    <td className="text-white">{detail.faction}</td>
                  </tr>
                )}
                {detail.size > 0 && detail.size !== 1 && (
                  <tr>
                    <th className="text-gray-400 pr-4 py-1">Size:</th>
                    <td className="text-white">{detail.size.toFixed(2)}</td>
                  </tr>
                )}
                {spawns.length > 0 && (
                  <tr>
                    <th className="text-gray-400 pr-4 py-1">Spawns:</th>
                    <td className="text-white">
                      {spawns.length} across {zones.length}{" "}
                      {zones.length === 1 ? "zone" : "zones"}
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </DetailSection>

          {/* Location map with spawn markers, per selected zone */}
          {zones.length > 0 && (
            <DetailSection title="Location">
              {/* Zone selector */}
              {zones.length > 1 && (
                <div className="flex flex-wrap gap-1.5 mb-2">
                  {zones.map((z) => (
                    <button
                      key={z.name}
                      onClick={() => setSelectedZone(z.name)}
                      className={`px-2 py-1 rounded text-xs border transition-colors ${
                        activeZone?.name === z.name
                          ? "border-wow-gold/60 bg-wow-gold/15 text-wow-gold"
                          : "border-gray-600/40 bg-white/[0.02] text-gray-300 hover:bg-white/5"
                      }`}
                    >
                      {z.name} <span className="text-gray-500">({z.pts.length})</span>
                    </button>
                  ))}
                </div>
              )}

              <div
                className="relative w-full aspect-[488/325] bg-cover bg-center rounded border border-white/20 shadow-lg cursor-pointer group overflow-hidden bg-black"
                style={{
                  backgroundImage: mapImage.src ? `url(${mapImage.src})` : "none",
                  maxWidth: "488px",
                }}
                onClick={() => mapImage.src && setShowMapModal(true)}
              >
                {!mapImage.src && !mapImage.loading && (
                  <div className="flex items-center justify-center h-full text-gray-500 text-sm">
                    No map for {activeZone?.name}
                  </div>
                )}
                {mapImage.loading && (
                  <div className="flex items-center justify-center h-full text-gray-500 text-sm animate-pulse">
                    Loading Map...
                  </div>
                )}
                {mapImage.src && renderMarkers(16)}
                {mapImage.src && (
                  <div className="absolute top-2 left-2 px-2 py-0.5 bg-black/70 rounded text-xs text-gray-300 border border-white/10">
                    {activeZone?.name} • {markers.length}{" "}
                    {markers.length === 1 ? "spawn" : "spawns"}
                  </div>
                )}
                <div className="absolute top-2 right-2 w-6 h-6 bg-black/50 rounded flex items-center justify-center text-white/80 opacity-0 group-hover:opacity-100 transition-opacity">
                  ⤢
                </div>
              </div>
            </DetailSection>
          )}

          {/* Related Quests */}
          {(startsQuests.length > 0 || endsQuests.length > 0) && (
            <DetailSection title="Related Quests">
              {startsQuests.length > 0 && (
                <div className="mb-4">
                  <h4 className="text-xs text-gray-500 uppercase mb-2">Starts</h4>
                  <div className="space-y-1">
                    {startsQuests.map((q) => (
                      <div
                        key={q.entry}
                        onClick={() => onNavigate("quest", q.entry)}
                        className="p-2 bg-white/[0.02] hover:bg-white/5 border-b border-white/5 cursor-pointer transition-colors"
                      >
                        <span className="text-wow-gold hover:text-yellow-300">
                          [{q.level}] {q.title}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
              {endsQuests.length > 0 && (
                <div>
                  <h4 className="text-xs text-gray-500 uppercase mb-2">Ends</h4>
                  <div className="space-y-1">
                    {endsQuests.map((q) => (
                      <div
                        key={q.entry}
                        onClick={() => onNavigate("quest", q.entry)}
                        className="p-2 bg-white/[0.02] hover:bg-white/5 border-b border-white/5 cursor-pointer transition-colors"
                      >
                        <span className="text-wow-gold hover:text-yellow-300">
                          [{q.level}] {q.title}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </DetailSection>
          )}
        </div>

        {/* Contains (Loot) */}
        {contains.length > 0 && (
          <DetailSection title={`Contains (${contains.length})`}>
            <LootGrid>
              {contains.map((item) => {
                const handlers = tooltipHook?.getItemHandlers?.(item.itemId) || {};
                return (
                  <LootItem
                    key={item.itemId}
                    item={{
                      entry: item.itemId,
                      name: item.name,
                      quality: item.quality,
                      iconPath: item.iconPath,
                      dropChance: `${item.chance.toFixed(1)}%`,
                    }}
                    onClick={() => onNavigate("item", item.itemId)}
                    showDropChance
                    {...handlers}
                  />
                );
              })}
            </LootGrid>
          </DetailSection>
        )}
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
                alt={activeZone?.name || "Location Map"}
                className="max-w-full max-h-[85vh] object-contain rounded-lg shadow-2xl"
              />
              {renderMarkers(20)}
            </div>
            {activeZone?.name && (
              <div className="absolute bottom-4 left-1/2 -translate-x-1/2 bg-black/80 px-4 py-2 rounded-lg text-white font-bold">
                {activeZone.name}
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

export default ObjectDetailView;
