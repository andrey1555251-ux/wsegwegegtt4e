package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/andrey1555251-ux/mindforge/internal/journal"
)

func TestJournalTrendEmptyWindow(t *testing.T) {
	withTempStore(t)
	out := captureStdout(t, func() error { return runJournalTrend([]string{"-days", "30"}) })
	if !strings.Contains(out, "no mood entries") {
		t.Errorf("expected empty hint, got:\n%s", out)
	}
}

func TestJournalTrendShowsHistogramAndAverage(t *testing.T) {
	withTempStore(t)
	st, err := open()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	now := time.Now()
	for i, mood := range []int{5, 4, 4, 3} {
		day := now.AddDate(0, 0, -i)
		if err := journal.Append(st, day, "x"); err != nil {
			t.Fatalf("append: %v", err)
		}
		if err := journal.SetMood(st, day, mood); err != nil {
			t.Fatalf("set mood: %v", err)
		}
	}
	out := captureStdout(t, func() error { return runJournalTrend([]string{"-days", "30"}) })
	if !strings.Contains(out, "Mood — last 30 days") {
		t.Errorf("missing header in:\n%s", out)
	}
	if !strings.Contains(out, "avg:") {
		t.Errorf("missing avg in:\n%s", out)
	}
	// (5+4+4+3)/4 = 4.00
	if !strings.Contains(out, "4.00") {
		t.Errorf("expected avg 4.00 in output:\n%s", out)
	}
	if !strings.Contains(out, "entries:") {
		t.Errorf("missing entries count in:\n%s", out)
	}
}
