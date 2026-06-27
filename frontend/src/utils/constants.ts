// Single source of truth for the remote WoW database base URL,
// used for external links and data scraping (not images).
export const DATABASE_BASE_URL = 'https://octowow.st/db'

// Default WoW client install dir. The client MPQs live under <base>\Data and are
// the source for on-demand model rendering. Persisted/overridden via the Tools
// page (localStorage "toolsBasePath"); this is the fallback when unset.
export const DEFAULT_WOW_BASE = 'C:\\WoW\\Octo'

// Resolve the configured client base path, falling back to the default.
export const getClientBasePath = (): string => {
    try {
        return (
            (typeof localStorage !== 'undefined' && localStorage.getItem('toolsBasePath')) ||
            DEFAULT_WOW_BASE
        )
    } catch {
        return DEFAULT_WOW_BASE
    }
}
