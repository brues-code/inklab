package parsers

import "bytes"

// Anti-bot interstitials — "Just a moment please...", "Verifying your browser" —
// are served with HTTP 200 in place of the page that was asked for. They matter
// to a parser because they carry no data: every parse of one succeeds and
// returns nothing, which reads exactly like a real page for an entity that has
// no spawns, no loot and no stats. Callers must therefore recognise them before
// parsing, and treat them as a failed fetch.

// challengeMarkers are byte sequences only an interstitial carries. Matching on
// body content rather than the `Server: BlazingFastWeb` header matters — that
// header stays on real pages served through the same proxy, so keying off it
// would reject everything.
var challengeMarkers = [][]byte{
	[]byte("bf.jquery"),
	[]byte("blzgfst-shark"),
	[]byte("protection by blazingfast"),
	[]byte("just a moment please"),
	[]byte("verifying your browser"),
	[]byte("checking your browser"),
}

// challengeScanBytes caps how much of a body is scanned. An interstitial is
// small (BlazingFast's is ~9.5 KB) and declares itself in its head, so this
// covers a whole challenge page while keeping the lowercase copy cheap during a
// bulk sync of real (much larger) pages.
const challengeScanBytes = 16 << 10

// IsChallengePage reports whether body is an anti-bot interstitial rather than a
// database page.
func IsChallengePage(body []byte) bool {
	head := body
	if len(head) > challengeScanBytes {
		head = head[:challengeScanBytes]
	}
	head = bytes.ToLower(head)
	for _, m := range challengeMarkers {
		if bytes.Contains(head, m) {
			return true
		}
	}
	return false
}
