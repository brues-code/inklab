package services

// DatabaseBaseURL is the single source of truth for the remote WoW database
// used for syncing items, spells, quests, NPCs and icons.
const DatabaseBaseURL = "https://octowow.st/db"

// scrapeConcurrency bounds how many octowow.st scrape requests run at once during
// full syncs. The default is deliberately low so shipped (non-dev) clients don't
// hammer the server; the dev build raises it via SetScrapeConcurrency. All the
// full-sync worker pools size themselves with scrapeWorkers().
var scrapeConcurrency = 2

// SetScrapeConcurrency sets the simultaneous-scrape cap (ignored if < 1). Called
// once at startup — dev builds bump it, shipped clients keep the low default.
func SetScrapeConcurrency(n int) {
	if n >= 1 {
		scrapeConcurrency = n
	}
}

// scrapeWorkers returns the configured scrape worker count (always >= 1).
func scrapeWorkers() int {
	if scrapeConcurrency < 1 {
		return 1
	}
	return scrapeConcurrency
}
