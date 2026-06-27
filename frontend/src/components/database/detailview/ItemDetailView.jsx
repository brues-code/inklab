import React, { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { tooltipQuery } from "../../../hooks/queries/tooltip";
import { useItemDetail, useItemFavorite } from "../../../hooks/queries/items";
import { queryClient } from "../../../queryClient";
import { ToggleFavorite } from "../../../services/api";
import {
  FixSingleItemIcon,
  SyncSingleItem,
} from "../../../../wailsjs/go/main/App";
import { getQualityColor, formatMoney, QUESTION_MARK_ICON } from "../../../utils/wow";
import { DATABASE_BASE_URL } from "../../../utils/constants";
import { useIcon } from "../../../services/useImage";
import {
  DetailPageLayout,
  DetailHeader,
  DetailSection,
  DetailLoading,
  DetailError,
  ItemTooltip,
  LootItem,
} from "../../ui";

// Helper component for Icon Header
const ItemIconHeader = ({
  iconName,
  iconPath,
  imgError,
  fixing,
  handleFixIcon,
  qualityColor,
}) => {
  // Determine icon name to use
  const name = iconPath || iconName;
  const icon = useIcon(name);

  // If explicit error state (from parent) or missing icon name
  const showFixButton = !name || imgError;

  if (showFixButton) {
    return (
      <button
        onClick={handleFixIcon}
        disabled={fixing}
        className="w-full h-full flex flex-col items-center justify-center bg-red-900/30 hover:bg-red-800/50 text-red-400 transition-colors gap-1"
        title={
          !name
            ? "No icon data - Click to fetch"
            : "Icon failed to load - Click to fix"
        }
      >
        <span className="text-2xl">{fixing ? "⏳" : "🔧"}</span>
        <span className="text-[10px]">{fixing ? "Fixing..." : "Fix Icon"}</span>
      </button>
    );
  }

  return (
    <>
      {icon.loading ? (
        <div className="w-full h-full bg-white/5 animate-pulse" />
      ) : (
        <img
          src={icon.src || QUESTION_MARK_ICON}
          className="w-full h-full object-cover"
          alt=""
        />
      )}
    </>
  );
};

// Small icon image resolved through the cached icon service.
const IconImg = ({ name, className }) => {
  const icon = useIcon(name);
  return (
    <img
      src={icon.src || QUESTION_MARK_ICON}
      className={className}
      alt=""
    />
  );
};

// A crafting reagent chip: icon + count badge + quality-colored name, linking to
// the reagent's item page with a hover tooltip.
const Reagent = ({ reagent, onNavigate, tooltipHook }) => {
  const handlers = tooltipHook?.getItemHandlers?.(reagent.entry) || {};
  return (
    <div
      className="flex items-center gap-1.5 pl-1 pr-2 py-1 bg-white/[0.03] hover:bg-white/[0.07] border border-white/5 rounded cursor-pointer transition-colors"
      onClick={() => onNavigate?.("item", reagent.entry)}
      {...handlers}
    >
      <div className="relative w-7 h-7 shrink-0">
        <IconImg name={reagent.iconPath} className="w-full h-full object-cover rounded border border-black/40" />
        {reagent.count > 1 && (
          <span className="absolute -bottom-0.5 -right-0.5 text-[10px] font-bold text-white bg-black/80 rounded px-0.5 leading-none py-0.5">
            {reagent.count}
          </span>
        )}
      </div>
      <span className="text-xs" style={{ color: getQualityColor(reagent.quality) }}>
        {reagent.name || `Item #${reagent.entry}`}
      </span>
    </div>
  );
};

// One recipe that produces this item: the craft spell (link + tooltip), its
// profession requirement and produced count, and the reagents consumed.
const CreatedBySource = ({ source, onNavigate, tooltipHook }) => (
  <div className="p-2.5 bg-white/[0.02] border border-white/5 rounded">
    <div className="flex items-center gap-2">
      <IconImg name={source.spellIcon} className="w-8 h-8 rounded border border-black/40 object-cover" />
      <span
        className="text-wow-rare font-bold cursor-pointer hover:underline"
        onClick={() => onNavigate?.("spell", source.spellId)}
        {...(tooltipHook?.getSpellHandlers?.(source.spellId) || {})}
      >
        {source.spellName}
      </span>
      {source.producedCount > 1 && (
        <span className="text-xs text-gray-500">creates {source.producedCount}</span>
      )}
      {source.skillName && (
        <span className="ml-auto text-xs text-gray-400 whitespace-nowrap">
          Requires {source.skillName}
          {source.reqSkill > 0 ? ` (${source.reqSkill})` : ""}
        </span>
      )}
    </div>
    {source.reagents?.length > 0 && (
      <div className="flex flex-wrap gap-1.5 mt-2">
        {source.reagents.map((rg) => (
          <Reagent key={rg.entry} reagent={rg} onNavigate={onNavigate} tooltipHook={tooltipHook} />
        ))}
      </div>
    )}
  </div>
);

const ItemDetailView = ({ entry, onBack, onNavigate, tooltipHook }) => {
  // The item's tooltip payload, from the shared Query cache (warmed by hover
  // elsewhere). The blanket invalidate in reloadData refetches this too.
  const { data: tooltip } = useQuery({ ...tooltipQuery(entry), enabled: !!entry });
  const { data: detail, isLoading: loading } = useItemDetail(entry);
  const [imgError, setImgError] = useState(false);
  const [fixing, setFixing] = useState(false);
  const [syncing, setSyncing] = useState(false);

  // Favorite status, cached per item and derived from the query (not state);
  // toggled optimistically in handleFavoriteToggle.
  const { data: isFavorite = false } = useItemFavorite(entry);

  // Reset the icon-error flag when the item changes (render-time, no effect).
  const [imgErrKey, setImgErrKey] = useState(entry);
  if (entry !== imgErrKey) {
    setImgErrKey(entry);
    setImgError(false);
  }

  // A sync or icon-fix can change this item anywhere it appears (the grid behind
  // this overlay, search, sets, loot tables, tooltips), so drop the whole cache;
  // active queries — this detail, its tooltip, the list — refetch immediately.
  const reloadData = async () => {
    await queryClient.invalidateQueries();
  };

  const handleFixIcon = async () => {
    setFixing(true);
    try {
      const result = await FixSingleItemIcon(entry);
      if (result.fixed > 0) {
        setImgError(false);
        await reloadData();
      } else {
        alert(
          `Auto-fetch failed: ${result.message}\n\n` +
            `This item's icon data could not be automatically retrieved.\n` +
            `Visit ${DATABASE_BASE_URL}/?item=${entry} to check if the item exists.`
        );
      }
    } catch (error) {
      alert(`Error: ${error}`);
    } finally {
      setFixing(false);
    }
  };

  const handleFavoriteToggle = async () => {
    let category = "";
    if (!isFavorite) {
        // If adding, ask for category (optional)
        const userInput = window.prompt("Enter a category for this favorite (optional):", "General");
        if (userInput === null) return; // Cancelled
        category = userInput;
    }
    
    try {
        const result = await ToggleFavorite(entry, category);
        if (result.success) {
            queryClient.setQueryData(["itemFavorite", entry], !isFavorite);
        } else {
            alert("Failed to toggle favorite: " + result.message);
        }
    } catch (err) {
        console.error("Favorite error:", err);
    }
  };

  // Sync full item data from turtlecraft.gg
  const handleSync = async () => {
    setSyncing(true);
    try {
      const result = await SyncSingleItem(entry);
      if (result && result.success) {
        setImgError(false);
        await reloadData();
      } else {
        alert(`Sync failed: ${result?.error || "Unknown error"}`);
      }
    } catch (error) {
      alert(`Sync error: ${error}`);
    } finally {
      setSyncing(false);
    }
  };

  const renderTooltipBlock = () => {
    if (!detail) return null;
    const dummyItem = {
      entry: detail.entry,
      quality: detail.quality,
      name: detail.name,
    };

    return (
      <div className="inline-block align-top min-w-[300px]">
        <ItemTooltip
          item={dummyItem}
          tooltip={tooltip}
          style={{ position: "static" }}
          interactive={true}
          onSpellClick={(spellId) => onNavigate?.("spell", spellId)}
          tooltipHook={tooltipHook}
        />
      </div>
    );
  };

  if (loading) return <DetailLoading />;
  
  if (!detail) {
     return (
        <DetailPageLayout onBack={onBack}>
           <div className="flex flex-col items-center justify-center p-20 text-gray-400 gap-6">
              <div className="text-xl">
                 Item <span className="text-white font-mono">{entry}</span> not found in local database.
              </div>
              <p className="text-sm text-gray-500 max-w-md text-center">
                 This item exists in the remote database reference but hasn't been synced to your local database yet.
              </p>
              <button 
                 onClick={handleSync} 
                 disabled={syncing}
                 className={`
                    px-6 py-3 bg-wow-gold text-black font-bold uppercase tracking-wider rounded 
                    hover:bg-yellow-400 disabled:opacity-50 disabled:cursor-not-allowed
                    shadow-[0_0_10px_rgba(255,209,0,0.2)] hover:shadow-[0_0_15px_rgba(255,209,0,0.4)]
                    transition-all
                 `}
              >
                 {syncing ? (
                    <span className="flex items-center gap-2">
                       <span className="animate-spin">↻</span> Syncing...
                    </span>
                 ) : (
                    "Sync from Remote"
                 )}
              </button>
           </div>
        </DetailPageLayout>
     );
  }

  const qualityColor = getQualityColor(detail.quality);

  return (
    <DetailPageLayout onBack={onBack}>
      <DetailHeader
        icon={
          <ItemIconHeader
            iconPath={detail.iconPath}
            imgError={imgError}
            fixing={fixing}
            handleFixIcon={handleFixIcon}
            qualityColor={qualityColor}
          />
        }
        iconBorderColor={qualityColor}
        title={detail.name}
        titleColor={qualityColor}
        subtitle={`Item Level ${detail.itemLevel}`}
        action={
          <div className="flex gap-2">
            <a
              href={`${DATABASE_BASE_URL}/?item=${entry}`}
              target="_blank"
              rel="noreferrer"
              className="px-3 py-1.5 text-xs font-bold uppercase rounded transition-colors bg-purple-700 hover:bg-purple-600 text-white"
              title="View on Turtle WoW Database"
            >
              🔗 OctoHead
            </a>
            <button
              onClick={() => {
                // Quality color codes (WoW format)
                const qualityColors = {
                  0: "ff9d9d9d", // Poor (grey)
                  1: "ffffffff", // Common (white)
                  2: "ff1eff00", // Uncommon (green)
                  3: "ff0070dd", // Rare (blue)
                  4: "ffa335ee", // Epic (purple)
                  5: "ffff8000", // Legendary (orange)
                  6: "ffe6cc80", // Artifact (gold)
                };
                const colorCode = qualityColors[detail.quality] || "ffffffff";
                // Format: |cCOLOR|Hitem:ID:0:0:0:0:0:0:0:0|h[NAME]|h|r
                // \124 is the escape for | in Lua
                // Escape quotes in name for Lua string
                const escapedName = detail.name.replace(/"/g, '\\"');
                const itemLink = `/script DEFAULT_CHAT_FRAME:AddMessage("\\124c${colorCode}\\124Hitem:${detail.entry}:0:0:0:0:0:0:0:0\\124h[${escapedName}]\\124h\\124r");`;
                navigator.clipboard
                  .writeText(itemLink)
                  .then(() =>
                    alert(
                      "In-game link copied to clipboard!\n\nPaste this in WoW chat to see the item link."
                    )
                  )
                  .catch((err) => alert("Failed to copy: " + err));
              }}
              className="px-3 py-1.5 text-xs font-bold uppercase rounded transition-colors bg-green-700 hover:bg-green-600 text-white"
              title="Copy in-game item link command to clipboard"
            >
              🔗 In-Game Link
            </button>
            <button
                onClick={handleFavoriteToggle}
                className={`px-3 py-1.5 text-xs font-bold uppercase rounded transition-colors flex items-center gap-1 ${
                    isFavorite 
                        ? "bg-red-600 hover:bg-red-500 text-white" 
                        : "bg-gray-700 hover:bg-gray-600 text-gray-300"
                }`}
                title={isFavorite ? "Remove from Favorites" : "Add to Favorites"}
            >
                {isFavorite ? "❤️ Favorited" : "🤍 Favorite"}
            </button>
            <button
              onClick={handleSync}
              disabled={syncing}
              className={`px-3 py-1.5 text-xs font-bold uppercase rounded transition-colors ${
                syncing
                  ? "bg-gray-600 text-gray-400 cursor-not-allowed"
                  : "bg-blue-600 hover:bg-blue-500 text-white"
              }`}
              title="Refresh item data from Turtle WoW Database"
            >
              {syncing ? "⏳ Syncing..." : "🔄 Sync"}
            </button>
          </div>
        }
      />

      <div className="flex flex-wrap gap-10">
        {/* Tooltip Block */}
        {renderTooltipBlock()}

        {/* Relations */}
        <div className="flex-1 min-w-[300px] space-y-8">
          {/* Created By */}
          {detail.createdBy?.length > 0 && (
            <DetailSection title="Created By">
              <div className="space-y-2">
                {detail.createdBy.map((source) => (
                  <CreatedBySource
                    key={source.spellId}
                    source={source}
                    onNavigate={onNavigate}
                    tooltipHook={tooltipHook}
                  />
                ))}
              </div>
            </DetailSection>
          )}

          {/* Contained In (reverse of Contains: chests / container items) */}
          {detail.containedIn?.length > 0 && (
            <DetailSection title="Contained In">
              <div className="space-y-1">
                {detail.containedIn.map((c) => (
                  <div
                    key={`${c.kind}-${c.entry}`}
                    className="flex items-center justify-between p-2 bg-white/[0.02] hover:bg-white/5 border-b border-white/5 cursor-pointer transition-colors"
                    onClick={() => onNavigate(c.kind, c.entry)}
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      {c.kind === "item" ? (
                        <IconImg
                          name={c.iconPath}
                          className="w-7 h-7 rounded border border-black/40 object-cover shrink-0"
                        />
                      ) : (
                        <span className="w-7 h-7 rounded bg-[#00B4FF]/20 border border-[#00B4FF]/40 text-[#00B4FF] text-[9px] font-bold flex items-center justify-center shrink-0">
                          OBJ
                        </span>
                      )}
                      <span
                        className="font-bold truncate hover:text-wow-gold"
                        style={{ color: c.kind === "item" ? getQualityColor(c.quality) : "#00B4FF" }}
                      >
                        {c.name}
                      </span>
                    </div>
                    {c.chance > 0 && (
                      <div className="text-wow-gold font-mono text-sm shrink-0">
                        {c.chance.toFixed(1)}%
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </DetailSection>
          )}

          {/* Dropped By */}
          {detail.droppedBy?.length > 0 && (
            <DetailSection title="Dropped By">
              <div className="space-y-1">
                {detail.droppedBy.map((npc) => (
                  <div
                    key={npc.entry}
                    className="flex items-center justify-between p-2 bg-white/[0.02] hover:bg-white/5 border-b border-white/5 cursor-pointer transition-colors"
                    onClick={() => onNavigate("npc", npc.entry)}
                  >
                    <div>
                      <div className="text-white font-bold hover:text-wow-gold">
                        {npc.name}
                      </div>
                      <div className="text-xs text-gray-500">
                        Level {npc.levelMin}
                        {npc.levelMax > npc.levelMin ? `-${npc.levelMax}` : ""}
                      </div>
                    </div>
                    <div className="text-wow-gold font-mono text-sm">
                      {npc.chance.toFixed(1)}%
                    </div>
                  </div>
                ))}
              </div>
            </DetailSection>
          )}

          {/* Sold By */}
          {detail.soldBy?.length > 0 && (
            <DetailSection title="Sold By">
              <div className="space-y-1">
                {detail.soldBy.map((npc) => {
                  const m = npc.cost > 0 ? formatMoney(npc.cost) : null;
                  return (
                    <div
                      key={npc.entry}
                      className="flex items-center justify-between p-2 bg-white/[0.02] hover:bg-white/5 border-b border-white/5 cursor-pointer transition-colors"
                      onClick={() => onNavigate("npc", npc.entry)}
                    >
                      <div>
                        <div className="text-white font-bold hover:text-wow-gold">
                          {npc.name}
                        </div>
                        <div className="text-xs text-gray-500">
                          Level {npc.levelMin}
                          {npc.levelMax > npc.levelMin ? `-${npc.levelMax}` : ""}
                          {npc.stock > 0 ? ` · ${npc.stock} in stock` : ""}
                        </div>
                      </div>
                      {m && (
                        <div className="text-sm font-mono whitespace-nowrap">
                          {m.g > 0 && <span className="text-yellow-400">{m.g}g </span>}
                          {(m.g > 0 || m.s > 0) && <span className="text-gray-300">{m.s}s </span>}
                          <span className="text-orange-400">{m.c}c</span>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            </DetailSection>
          )}

          {/* Reward From */}
          {detail.rewardFrom?.length > 0 && (
            <DetailSection title="Reward From">
              <div className="space-y-1">
                {detail.rewardFrom.map((q) => (
                  <div
                    key={q.entry}
                    className="flex items-center gap-3 p-2 bg-white/[0.02] hover:bg-white/5 border-b border-white/5 cursor-pointer transition-colors"
                    onClick={() => onNavigate("quest", q.entry)}
                  >
                    <div className="flex-1 min-w-0">
                      <div className="text-wow-gold font-bold truncate">
                        {q.title}
                      </div>
                      <div className="text-xs text-gray-500">
                        Level {q.level}
                        {q.isChoice && (
                          <span className="ml-2 text-[10px] border border-white/10 px-1 rounded uppercase">
                            Choice
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </DetailSection>
          )}

          {/* Starts Quest */}
          {detail.startsQuest && (
            <DetailSection title="Starts Quest">
              <div
                className="flex items-center gap-3 p-2 bg-white/[0.02] hover:bg-white/5 border-b border-white/5 cursor-pointer transition-colors"
                onClick={() => onNavigate("quest", detail.startsQuest.entry)}
              >
                <div className="flex-1 min-w-0">
                  <div className="text-wow-gold font-bold truncate">
                    {detail.startsQuest.title}
                  </div>
                  <div className="text-xs text-gray-500">
                    Level {detail.startsQuest.level}
                  </div>
                </div>
              </div>
            </DetailSection>
          )}

          {/* Objective Of */}
          {detail.objectiveOf?.length > 0 && (
            <DetailSection title="Objective Of">
              <div className="space-y-1">
                {detail.objectiveOf.map((q) => (
                  <div
                    key={q.entry}
                    className="flex items-center gap-3 p-2 bg-white/[0.02] hover:bg-white/5 border-b border-white/5 cursor-pointer transition-colors"
                    onClick={() => onNavigate("quest", q.entry)}
                  >
                    <div className="flex-1 min-w-0">
                      <div className="text-wow-gold font-bold truncate">{q.title}</div>
                      <div className="text-xs text-gray-500">Level {q.level}</div>
                    </div>
                  </div>
                ))}
              </div>
            </DetailSection>
          )}

          {/* Contains */}
          {detail.contains?.length > 0 && (
            <DetailSection title="Contains">
              <div className="grid grid-cols-1 gap-1">
                {detail.contains.map((item) => (
                  <LootItem
                    key={item.entry}
                    item={{
                      ...item,
                      dropChance: item.chance
                        ? item.chance.toFixed(1) + "%"
                        : null,
                    }}
                    showDropChance={true}
                    onClick={() => onNavigate("item", item.entry)}
                  />
                ))}
              </div>
            </DetailSection>
          )}
        </div>
      </div>
    </DetailPageLayout>
  );
};

export default ItemDetailView;
