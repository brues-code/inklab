import React, { useState, useEffect } from "react";
import { DATABASE_BASE_URL } from "../../../utils/constants";
import { GetFactionDetail } from "../../../../wailsjs/go/main/App";
import {
  DetailPageLayout,
  DetailHeader,
  DetailSection,
  DetailLoading,
  DetailError,
} from "../../ui";

const FactionDetailView = ({ id, onBack, onNavigate }) => {
  const [detail, setDetail] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    GetFactionDetail(id).then((res) => {
      setDetail(res);
      setLoading(false);
    });
  }, [id]);

  if (loading) return <DetailLoading />;
  if (!detail) return <DetailError message="Faction not found" onBack={onBack} />;

  const quests = detail.quests || [];
  const questGivers = detail.questGivers || [];
  const members = detail.members || [];

  const npcGrid = (npcs) => (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2">
      {npcs.map((n) => (
        <div
          key={n.entry}
          onClick={() => onNavigate("npc", n.entry)}
          className="p-3 flex items-center justify-between gap-2 bg-white/[0.02] hover:bg-white/5 border border-white/5 rounded cursor-pointer transition-colors"
        >
          <span className="text-wow-gold hover:text-yellow-300 font-medium truncate">
            {n.name}
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

      {/* Quests that reward reputation */}
      {quests.length > 0 && (
        <DetailSection title={`Reputation Quests (${quests.length})`}>
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
        </DetailSection>
      )}

      {/* NPCs that give this faction's reputation quests */}
      {questGivers.length > 0 && (
        <DetailSection title={`Quest Givers (${questGivers.length})`}>
          {npcGrid(questGivers)}
        </DetailSection>
      )}

      {/* NPCs belonging to this faction (from FactionTemplate) */}
      {members.length > 0 && (
        <DetailSection title={`Faction Members (${members.length})`}>
          {npcGrid(members)}
        </DetailSection>
      )}
    </DetailPageLayout>
  );
};

export default FactionDetailView;
