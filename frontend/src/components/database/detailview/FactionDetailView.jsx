import React, { useState } from "react";
import { DATABASE_BASE_URL } from "../../../utils/constants";
import { useFactionDetail } from "../../../hooks/queries/factions";
import {
  DetailPageLayout,
  DetailHeader,
  DetailSection,
  DetailLoading,
  DetailError,
} from "../../ui";

const FactionDetailView = ({ id, onBack, onNavigate }) => {
  const [activeTab, setActiveTab] = useState(null);
  const { data: detail, isLoading: loading } = useFactionDetail(id);

  if (loading) return <DetailLoading />;
  if (!detail) return <DetailError message="Faction not found" onBack={onBack} />;

  const quests = detail.quests || [];
  const questGivers = detail.questGivers || [];
  const members = detail.members || [];

  // Tabs for the relationship tables (only those with data).
  const tabs = [
    quests.length > 0 && { id: "quests", label: `Reputation Quests (${quests.length})` },
    questGivers.length > 0 && { id: "givers", label: `Quest Givers (${questGivers.length})` },
    members.length > 0 && { id: "members", label: `Faction Members (${members.length})` },
  ].filter(Boolean);
  const currentTab = tabs.some((t) => t.id === activeTab) ? activeTab : tabs[0]?.id;

  const npcGrid = (npcs) => (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2">
      {npcs.map((n) => (
        <div
          key={n.entry}
          onClick={() => onNavigate("npc", n.entry)}
          className="p-3 flex items-center justify-between gap-2 bg-white/[0.02] hover:bg-white/5 border border-white/5 rounded cursor-pointer transition-colors"
        >
          <span className="min-w-0 truncate">
            <span className="text-wow-gold hover:text-yellow-300 font-medium">
              {n.name}
            </span>
            {n.subname && (
              <span className="text-gray-500 text-xs ml-1">&lt;{n.subname}&gt;</span>
            )}
          </span>
          <span className="text-xs text-gray-500 whitespace-nowrap">
            Lvl {n.levelMin}
            {n.levelMax > n.levelMin ? `-${n.levelMax}` : ""}
          </span>
        </div>
      ))}
    </div>
  );

  // Side color and icon
  const getSideStyle = () => {
    switch (detail.side) {
      case 1:
        return { color: "text-blue-400", icon: "🔵", img: "/Alliance_15.webp", name: "Alliance" };
      case 2:
        return { color: "text-red-400", icon: "🔴", img: "/Horde_15.webp", name: "Horde" };
      default:
        return { color: "text-yellow-400", icon: "🟡", img: "/Neutral_15.webp", name: "Neutral" };
    }
  };

  const sideStyle = getSideStyle();

  return (
    <DetailPageLayout onBack={onBack}>
      <DetailHeader
        icon={
          <div className="w-full h-full flex items-center justify-center bg-gray-900 border border-gray-700 p-1">
             <img 
                src={sideStyle.img} 
                alt={sideStyle.name} 
                className="w-full h-full object-contain" 
             />
          </div>
        }
        iconBorderColor={sideStyle.color}
        title={detail.name}
        titleColor={sideStyle.color}
        subtitle={sideStyle.name}
        action={
          <a
            href={`${DATABASE_BASE_URL}/?faction=${detail.id}`}
            target="_blank"
            rel="noreferrer"
            className="px-3 py-1.5 text-xs font-bold uppercase rounded transition-colors bg-purple-700 hover:bg-purple-600 text-white"
          >
            🔗 OctoHead
          </a>
        }
      />

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Description */}
        {detail.description && (
          <DetailSection title="Description">
            <p className="text-gray-300 text-sm leading-relaxed">
              {detail.description}
            </p>
          </DetailSection>
        )}

        {/* Quick Facts */}
        <DetailSection title="Quick Facts">
          <table className="infobox-table text-sm w-full">
            <tbody>
              <tr>
                <th className="text-gray-400 pr-4 py-1">Faction ID:</th>
                <td className="text-white">{detail.id}</td>
              </tr>
              <tr>
                <th className="text-gray-400 pr-4 py-1">Side:</th>
                <td className={sideStyle.color}>{sideStyle.name}</td>
              </tr>
              <tr>
                <th className="text-gray-400 pr-4 py-1">Related Quests:</th>
                <td className="text-white">{quests.length}</td>
              </tr>
            </tbody>
          </table>
        </DetailSection>
      </div>

      {/* Relationship tables — tabbed */}
      {tabs.length > 0 && (
        <div className="mt-8">
          <div className="border-b border-white/20 mb-4 flex gap-1">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`px-4 py-2 text-sm font-bold transition-all relative top-[1px] ${
                  currentTab === tab.id
                    ? "tab-btn-active text-white border-b-2 border-wow-gold"
                    : "tab-btn-inactive text-gray-400 hover:text-gray-200"
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>

          <div className="animate-fade-in">
            {currentTab === "quests" && (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2">
                {quests.map((q) => (
                  <div
                    key={q.entry}
                    onClick={() => onNavigate("quest", q.entry)}
                    className="p-3 bg-white/[0.02] hover:bg-white/5 border border-white/5 rounded cursor-pointer transition-colors"
                  >
                    <span className="text-wow-gold hover:text-yellow-300 font-medium">
                      [{q.level}] {q.title}
                    </span>
                  </div>
                ))}
              </div>
            )}
            {currentTab === "givers" && npcGrid(questGivers)}
            {currentTab === "members" && npcGrid(members)}
          </div>
        </div>
      )}
    </DetailPageLayout>
  );
};

export default FactionDetailView;
