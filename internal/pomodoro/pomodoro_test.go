package pomodoro

import (
	"testing"
	"time"

	"github.com/andrey1555251-ux/mindforge/internal/store"
)

func TestPhaseString(t *testing.T) {
	cases := []struct {
		p    Phase
		want string
	}{
		{PhaseFocus, "focus"},
		{PhaseShortBreak, "short break"},
		{PhaseLongBreak, "long break"},
		{Phase(99), "?"},
	}
	for _, c := range cases {
		if got := c.p.String(); got != c.want {
			t.Errorf("Phase(%d).String() = %q, want %q", c.p, got, c.want)
		}
	}
}

func TestWithDefaultsFillsBlanks(t *testing.T) {
	settings := store.Settings{
		PomodoroMinutes:    25,
		ShortBreakMinutes:  5,
		LongBreakMinutes:   15,
		PomodorosUntilLong: 4,
	}
	o := Options{}.WithDefaults(settings)
	if o.Rounds != 1 {
		t.Errorf("Rounds = %d, want 1", o.Rounds)
	}
	if o.FocusMinutes != 25 {
		t.Errorf("FocusMinutes = %d, want 25", o.FocusMinutes)
	}
	if o.ShortBreak != 5 {
		t.Errorf("ShortBreak = %d, want 5", o.ShortBreak)
	}
	if o.LongBreak != 15 {
		t.Errorf("LongBreak = %d, want 15", o.LongBreak)
	}
	if o.UntilLong != 4 {
		t.Errorf("UntilLong = %d, want 4", o.UntilLong)
	}
}

func TestWithDefaultsKeepsOverrides(t *testing.T) {
	settings := store.Settings{
		PomodoroMinutes:    25,
		ShortBreakMinutes:  5,
		LongBreakMinutes:   15,
		PomodorosUntilLong: 4,
	}
	o := Options{
		Rounds:       3,
		FocusMinutes: 50,
		ShortBreak:   10,
		LongBreak:    20,
		UntilLong:    2,
	}.WithDefaults(settings)
	if o.Rounds != 3 || o.FocusMinutes != 50 || o.ShortBreak != 10 || o.LongBreak != 20 || o.UntilLong != 2 {
		t.Errorf("overrides not preserved: %+v", o)
	}
}

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		seconds int
		want    string
	}{
		{0, "00:00"},
		{59, "00:59"},
		{60, "01:00"},
		{125, "02:05"},
		{60 * 25, "25:00"},
	}
	for _, c := range cases {
		got := formatDuration(time.Duration(c.seconds) * time.Second)
		if got != c.want {
			t.Errorf("formatDuration(%d) = %q, want %q", c.seconds, got, c.want)
		}
	}
}
