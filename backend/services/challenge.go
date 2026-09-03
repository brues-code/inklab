package services

import "errors"

// ErrChallenge marks a response that is an anti-bot interstitial rather than the
// page we asked for. It needs its own sentinel because these pages arrive with
// HTTP 200: octowow.st sits behind BlazingFast, whose holding page carries no
// database content, so a status check alone accepts it as a good page. Every
// parser then finds nothing and the scrape looks like a success that honestly
// returned no rows — which is how an object sync could delete a node's spawns
// and write nothing back. Definitive, never retried: only running the page's own
// JS clears it. Detection lives in parsers.IsChallengePage.
var ErrChallenge = errors.New("the database site served an anti-bot challenge page instead of data")
