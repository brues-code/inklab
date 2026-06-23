import React, { useState, useEffect } from "react";
import { DATABASE_BASE_URL } from "../../../utils/constants";
import { GetObjectDetail } from "../../../../wailsjs/go/main/App";
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
  const [detail, setDetail] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    GetObjectDetail(entry).then((res) => {
      setDetail(res);
      setLoading(false);
    });
  }, [entry]);

  if (loading) return <DetailLoading />;
  if (!detail) return <DetailError message="Object not found" onBack={onBack} />;

  const startsQuests = detail.startsQuests || [];
  const endsQuests = detail.endsQuests || [];
  const contains = detail.contains || [];

  return (
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
          <a
            href={`${DATABASE_BASE_URL}/?object=${detail.entry}`}
            target="_blank"
            rel="noreferrer"
            className="px-3 py-1.5 text-xs font-bold uppercase rounded transition-colors bg-purple-700 hover:bg-purple-600 text-white"
          >
            🔗 TurtleCraft
          </a>
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
            </tbody>
          </table>
        </DetailSection>

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
  );
};

export default ObjectDetailView;
