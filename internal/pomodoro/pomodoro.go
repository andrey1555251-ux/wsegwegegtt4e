// Package pomodoro implements a tiny pomodoro timer.  It runs the
// countdown in the foreground, drawing a progress bar in-place and
// finishing with a beep so it works in any terminal — including the
// most ascetic Windows console.
package pomodoro

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/andrey1555251-ux/mindforge/internal/store"
	"github.com/andrey1555251-ux/mindforge/internal/ui"
)

// Phase is one segment of a pomodoro session.
type Phase int

const (
	PhaseFocus Phase = iota
	PhaseShortBreak
	PhaseLongBreak
)

func (p Phase) String() string {
	switch p {
	case PhaseFocus:
		return "focus"
	case PhaseShortBreak:
		return "short break"
	case PhaseLongBreak:
		return "long break"
	}
	return "?"
}

// Options describe one run of `mindforge pomodoro`.
type Options struct {
	Label        string
	Rounds       int
	FocusMinutes int
	ShortBreak   int
	LongBreak    int
	UntilLong    int
	Quiet        bool
}

// FromSettings fills missing options with the user's stored preferences.
func (o Options) WithDefaults(s store.Settings) Options {
	if o.Rounds <= 0 {
		o.Rounds = 1
	}
	if o.FocusMinutes <= 0 {
		o.FocusMinutes = s.PomodoroMinutes
	}
	if o.ShortBreak <= 0 {
		o.ShortBreak = s.ShortBreakMinutes
	}
	if o.LongBreak <= 0 {
		o.LongBreak = s.LongBreakMinutes
	}
	if o.UntilLong <= 0 {
		o.UntilLong = s.PomodorosUntilLong
	}
	return o
}

// Run starts a pomodoro session with the given options writing live
// updates to w.  Pressing Ctrl+C during a focus phase records an
// interrupted log so streaks remain honest.
func Run(ctx context.Context, w io.Writer, st *store.Store, opts Options) error {
	opts = opts.WithDefaults(snapshotSettings(st))

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	completedFocus := 0
	for round := 1; round <= opts.Rounds; round++ {
		// Focus phase.
		fmt.Fprintf(w, "\n%s round %d / %d  ·  %s\n", ui.Magenta("Focus"), round, opts.Rounds, ui.Dim(opts.Label))
		startedAt := time.Now()
		ok := runCountdown(ctx, w, signals, time.Duration(opts.FocusMinutes)*time.Minute, ui.Magenta)
		finishedAt := time.Now()
		log := store.PomodoroLog{
			StartedAt:   startedAt,
			FinishedAt:  finishedAt,
			Minutes:     opts.FocusMinutes,
			Label:       opts.Label,
			Interrupted: !ok,
		}
		_ = st.Use(func(d *store.Data) error {
			d.Pomodoros = append(d.Pomodoros, log)
			return nil
		})
		if !ok {
			fmt.Fprintln(w, ui.Yellow("\ninterrupted — see you next time"))
			return nil
		}
		completedFocus++
		if !opts.Quiet {
			beep(w)
		}
		fmt.Fprintln(w, ui.Green("done!  great work."))

		// Decide which break to take.
		if round == opts.Rounds {
			break
		}
		var breakMin int
		var breakName string
		if completedFocus%opts.UntilLong == 0 {
			breakMin = opts.LongBreak
			breakName = "long break"
		} else {
			breakMin = opts.ShortBreak
			breakName = "short break"
		}
		fmt.Fprintf(w, "\n%s  ·  %d minutes\n", ui.Cyan(breakName), breakMin)
		ok = runCountdown(ctx, w, signals, time.Duration(breakMin)*time.Minute, ui.Cyan)
		if !ok {
			return nil
		}
		if !opts.Quiet {
			beep(w)
		}
	}
	return nil
}

func snapshotSettings(s *store.Store) store.Settings {
	var out store.Settings
	s.View(func(d *store.Data) { out = d.Settings })
	return out
}

func runCountdown(ctx context.Context, w io.Writer, sig <-chan os.Signal, total time.Duration, color func(string) string) bool {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	end := time.Now().Add(total)

	width := 30
	for {
		remaining := time.Until(end)
		if remaining <= 0 {
			drawBar(w, width, 1.0, total, total, color)
			fmt.Fprintln(w)
			return true
		}
		drawBar(w, width, 1-float64(remaining)/float64(total), total-remaining, total, color)
		select {
		case <-ctx.Done():
			fmt.Fprintln(w)
			return false
		case <-sig:
			fmt.Fprintln(w)
			return false
		case <-tick.C:
		}
	}
}

func drawBar(w io.Writer, width int, fraction float64, elapsed, total time.Duration, color func(string) string) {
	bar := ui.Progress(width, fraction)
	fmt.Fprintf(w, "\r%s  %s / %s ", color(bar), formatDuration(elapsed), formatDuration(total))
	if f, ok := w.(interface{ Sync() error }); ok {
		_ = f.Sync()
	}
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	m := int(d / time.Minute)
	s := int((d % time.Minute) / time.Second)
	return fmt.Sprintf("%02d:%02d", m, s)
}

func beep(w io.Writer) {
	// \a is the ASCII bell — supported by virtually every terminal,
	// including Windows console.  We also flash a tiny "ding" so
	// users with the bell muted still notice.
	fmt.Fprint(w, "\a")
	fmt.Fprintln(w, strings.TrimSpace("  *ding*"))
}
