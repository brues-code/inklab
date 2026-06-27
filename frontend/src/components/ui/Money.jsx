import { formatMoney } from '../../utils/wow'
import { useImage } from '../../services/useImage'

// Coin colors for the fallback (match the sell-price colors used elsewhere).
const COIN_COLOR = { gold: '#FFD700', silver: '#C0C0C0', copper: '#B87333' }

// One coin denomination: the icon is cropped from the client's UI-MoneyIcons
// sprite during the client import (data/coin_icons) and served via the image
// service. Before a client import (no icon) it falls back to a coin-colored
// circle rather than the questionmark placeholder.
const Coin = ({ denom, amount }) => {
    const { src } = useImage('coin', denom)
    return (
        <span className="inline-flex items-center gap-0.5 text-white">
            {amount}
            {src ? (
                <img src={src} alt={denom} className="inline-block h-3.5 w-3.5" />
            ) : (
                <span
                    title={denom}
                    className="inline-block h-3 w-3 rounded-full border border-black/40"
                    style={{ backgroundColor: COIN_COLOR[denom] }}
                />
            )}
        </span>
    )
}

/**
 * Renders a copper amount as gold/silver/copper with the in-game coin icons,
 * omitting zero denominations (Wowhead-style) — e.g. 250 -> "2s 50c".
 * Renders nothing for 0/undefined.
 */
export const Money = ({ copper, className = '' }) => {
    const m = formatMoney(copper)
    const parts = []
    if (m.g > 0) parts.push(['gold', m.g])
    if (m.s > 0) parts.push(['silver', m.s])
    if (m.c > 0) parts.push(['copper', m.c])
    if (!parts.length) return null

    return (
        <span className={`inline-flex items-center gap-1.5 ${className}`}>
            {parts.map(([denom, amount]) => (
                <Coin key={denom} denom={denom} amount={amount} />
            ))}
        </span>
    )
}

export default Money
