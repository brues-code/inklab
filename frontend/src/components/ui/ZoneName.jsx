import React, { useMemo } from 'react'
import { useZoneNames } from '../../hooks/queries/zones'

// zoneKey mirrors the backend's zoneKey (zone_repository.go): lowercase, drop
// parenthetical/bracket segments, keep only alphanumerics, strip a leading
// "the", then apply alias fixes. Used to match a raw spawn/folder zone string to
// the official display name returned by GetZoneNames.
const zoneAliases = { ogrimmar: 'orgrimmar' }
const zoneKey = (s) => {
    if (!s) return ''
    let k = s
        .toLowerCase()
        .replace(/[([][^)\]]*[)\]]/g, '') // drop "(Dungeon)" / "[...]"
        .replace(/[^a-z0-9]/g, '')
    if (k.startsWith('the')) k = k.slice(3)
    return zoneAliases[k] || k
}

// humanize is the fallback when a name has no AreaTable match: split camelCase /
// letter-digit boundaries so at least "DunMorogh" -> "Dun Morogh".
const humanize = (s) =>
    (s || '').replace(/([a-z0-9])([A-Z])/g, '$1 $2').replace(/([A-Za-z])([0-9])/g, '$1 $2')

/**
 * ZoneName is the single place zone names get rendered. It resolves a raw
 * spawn/folder zone string (or a comma-separated list of them) to the official
 * localized AreaTable name. When onNavigate is given, each resolved zone links to
 * its zone page. Falls back to a camelCase-split of the raw string if no match.
 */
export function ZoneName({ name, onNavigate, className, fallback = null }) {
    const { data } = useZoneNames()
    const byKey = useMemo(() => {
        const m = new Map()
        for (const z of data || []) m.set(z.key, z)
        return m
    }, [data])

    const parts = (name || '')
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean)

    if (parts.length === 0) {
        return fallback ? <span className={className}>{fallback}</span> : null
    }

    return (
        <span className={className}>
            {parts.map((p, i) => {
                const hit = byKey.get(zoneKey(p))
                const label = hit ? hit.name : humanize(p)
                return (
                    <React.Fragment key={`${p}-${i}`}>
                        {i > 0 && ', '}
                        {hit && onNavigate ? (
                            <span
                                className="cursor-pointer hover:underline"
                                onClick={(e) => {
                                    e.stopPropagation()
                                    onNavigate('zone', hit.id)
                                }}
                            >
                                {label}
                            </span>
                        ) : (
                            label
                        )}
                    </React.Fragment>
                )
            })}
        </span>
    )
}

export default ZoneName
