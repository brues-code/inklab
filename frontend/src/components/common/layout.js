/**
 * Shared layout configuration for DatabasePage and AtlasLootPage
 */

// Grid column layout for 4-column pages (Class, SubClass, Filters, List)
// Filter width: 300px for more space
export const GRID_LAYOUT = '180px 180px 300px 1fr'
export const GRID_LAYOUT_NO_FILTER = '180px 180px 1fr'

// Grid column layout for Items tab (5 columns: Class, SubClass, Slot, Filters, List)
export const ITEMS_LAYOUT = '180px 180px 180px 300px 1fr'
export const ITEMS_LAYOUT_NO_FILTER = '180px 180px 180px 1fr'
// Sets tab: Class | Item Sets | Set detail
export const SETS_LAYOUT = '170px 300px 1fr'

// Individual column widths (for reference or future use)
export const COLUMN_WIDTHS = {
    FIRST: '180px',
    SECOND: '180px',
    THIRD: '200px',
    FILTERS: '300px',
    FOURTH: '1fr',
}
