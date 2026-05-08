package cli

import (
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/andrey1555251-ux/mindforge/internal/habits"
	"github.com/andrey1555251-ux/mindforge/internal/stats"
	"github.com/andrey1555251-ux/mindforge/internal/tasks"
	"github.com/andrey1555251-ux/mindforge/internal/ui"
)

// runMotd is a tiny banner suitable for putting in your shell rc.  It
// prints in two lines so it doesn't take over the terminal.
func runMotd(w io.Writer) error {
	st, err := open()
	if err != nil {
		return err
	}
	d, err := st.Snapshot()
	if err != nil {
		return err
	}
	now := time.Now()
	s := stats.Compute(d, now)
	open := tasks.List(st, tasks.FilterOpts{OverdueOnly: true})
	soon := tasks.List(st, tasks.FilterOpts{DueSoonHrs: 24})
	hs := habits.All(st)
	pendingHabits := 0
	for _, h := range hs {
		if !habits.CheckedOn(h, now) {
			pendingHabits++
		}
	}
	greeting := timeOfDayGreeting(now)
	fmt.Fprintf(w, "%s — %s · %s open · %s overdue · %s due soon · %s habit(s) waiting\n",
		ui.Bold(ui.Cyan(greeting)),
		now.Format("Mon 02 Jan"),
		ui.Bold(itoa(s.OpenTasks)),
		ui.Red(itoa(len(open))),
		ui.Yellow(itoa(len(soon))),
		ui.Yellow(itoa(pendingHabits)))
	fmt.Fprintf(w, "%s focus today · %s pomodoros · %s journal streak\n",
		ui.Bold(formatMinutes(s.FocusMinutesToday)),
		ui.Bold(itoa(s.PomodorosToday)),
		streakBadge(s.JournalStreak))
	if !d.Settings.DisableMotivation {
		fmt.Fprintln(w, ui.Italic(ui.Magenta(randomQuote())))
	}
	return nil
}

// timeOfDayGreeting picks the right "good X" salutation.
func timeOfDayGreeting(now time.Time) string {
	switch h := now.Hour(); {
	case h < 5:
		return "good night"
	case h < 12:
		return "good morning"
	case h < 17:
		return "good afternoon"
	case h < 22:
		return "good evening"
	default:
		return "good night"
	}
}

// Sort here is unused but the import for `sort` keeps the package tidy
// against future extensions like top-N pinned notes.  We keep it only
// if needed by future code; otherwise the build will warn.
var _ = sort.Strings
