package cli

import (
	"math/rand"
	"time"
)

// quotes is a small hand-curated set of friendly nudges used by
// `mindforge today` and `mindforge quote`.  They are intentionally
// short so they fit on one terminal line.
var quotes = []string{
	"start small.  finishing one thing beats starting ten.",
	"the work that matters is the work you do today.",
	"your future self will thank you for the next 25 minutes.",
	"a tiny note today is more useful than a perfect one tomorrow.",
	"focus is a muscle — flex it kindly.",
	"the page is patient.  begin anywhere.",
	"do less, but better.",
	"deep work compounds.",
	"breathe.  one task at a time.",
	"discipline is just showing up again.",
	"ideas are cheap.  notes are free.  shipping is everything.",
	"keep going.  the streak is on your side.",
	"clarity comes from action, not the other way around.",
	"the best time to start was yesterday.  the next best time is now.",
	"slow is smooth, smooth is fast.",
}

var quoteRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func randomQuote() string {
	return quotes[quoteRand.Intn(len(quotes))]
}
