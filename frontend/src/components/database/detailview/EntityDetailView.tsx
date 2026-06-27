import {
    NPCDetailView,
    QuestDetailView,
    ItemDetailView,
    SpellDetailView,
    ObjectDetailView,
    ZoneDetailView,
    FactionDetailView,
} from './index'

/**
 * Shared entity detail renderer: a Back bar + the correct detail view for the
 * given entity `type`. Both the Database and AtlasLoot detail routes render
 * this, replacing the duplicated detail-overlay blocks that previously lived in
 * DatabasePage and AtlasLootPage.
 *
 * `onBack` should call router history back, and `onNavigate(type, entry)` should
 * push a sibling detail route, so the browser Back button walks the trail.
 */
type Props = {
    type: string
    entry: number
    onBack: () => void
    onNavigate: (type: string, entry: number) => void
    tooltipHook: any
}

export function EntityDetailView({ type, entry, onBack, onNavigate, tooltipHook }: Props) {
    return (
        <div className="flex h-full flex-1 flex-col overflow-hidden">
            {/* Detail header with breadcrumb */}
            <div className="flex items-center gap-4 border-b border-border-dark bg-bg-hover px-4 py-2">
                <button
                    onClick={onBack}
                    className="rounded border border-border-light bg-bg-panel px-4 py-1.5 text-sm text-gray-400 transition-colors hover:bg-bg-active hover:text-white"
                >
                    ← Back
                </button>
                <span className="text-sm text-gray-500">
                    Viewing: <b className="uppercase text-gray-300">{type}</b>
                    <span className="ml-2 rounded bg-black/20 px-1.5 py-0.5 font-mono">
                        #{entry}
                    </span>
                </span>
            </div>

            {/* Detail content */}
            <div className="flex-1 overflow-auto">
                {type === 'npc' && (
                    <NPCDetailView
                        entry={entry}
                        onNavigate={onNavigate}
                        onBack={onBack}
                        tooltipHook={tooltipHook}
                    />
                )}
                {type === 'quest' && (
                    <QuestDetailView
                        entry={entry}
                        onNavigate={onNavigate}
                        onBack={onBack}
                        tooltipHook={tooltipHook}
                    />
                )}
                {type === 'item' && (
                    <ItemDetailView
                        entry={entry}
                        onNavigate={onNavigate}
                        onBack={onBack}
                        tooltipHook={tooltipHook}
                    />
                )}
                {type === 'spell' && (
                    <SpellDetailView
                        entry={entry}
                        onNavigate={onNavigate}
                        onBack={onBack}
                        tooltipHook={tooltipHook}
                    />
                )}
                {type === 'object' && (
                    <ObjectDetailView
                        entry={entry}
                        onNavigate={onNavigate}
                        onBack={onBack}
                        tooltipHook={tooltipHook}
                    />
                )}
                {type === 'zone' && (
                    <ZoneDetailView entry={entry} onNavigate={onNavigate} onBack={onBack} />
                )}
                {type === 'faction' && (
                    <FactionDetailView id={entry} onNavigate={onNavigate} onBack={onBack} />
                )}
            </div>
        </div>
    )
}

export default EntityDetailView
