package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/andrey1555251-ux/mindforge/internal/habits"
	"github.com/andrey1555251-ux/mindforge/internal/journal"
	"github.com/andrey1555251-ux/mindforge/internal/store"
	"github.com/andrey1555251-ux/mindforge/internal/tasks"
)

func seedSummaryFixture(t *testing.T) *store.Store {
	t.Helper()
	withTempStore(t)
	st, err := open()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	now := time.Now()

	// 1 completed task and 1 overdue open task.
	if _, err := tasks.Add(st, "shipped", 1, nil, []string{"work"}, ""); err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := tasks.Done(st, 1); err != nil {
		t.Fatalf("done: %v", err)
	}
	yesterday := now.AddDate(0, 0, -1)
	if _, err := tasks.Add(st, "late", 2, &yesterday, nil, ""); err != nil {
		t.Fatalf("add overdue: %v", err)
	}

	// 1 pomodoro, with a label, recorded "now".
	_ = st.Use(func(d *store.Data) error {
		d.Pomodoros = append(d.Pomodoros, store.PomodoroLog{
			StartedAt:  now.Add(-30 * time.Minute),
			FinishedAt: now,
			Minutes:    25,
			Label:      "deep work",
		})
		return nil
	})

	// Journal entry today with mood.
	if err := journal.Append(st, now, "shipped MindForge"); err != nil {
		t.Fatalf("journal append: %v", err)
	}
	if err := journal.SetMood(st, now, 4); err != nil {
		t.Fatalf("set mood: %v", err)
	}

	// Habit checked today.
	id, err := habits.Add(st, "drink water", 1)
	if err != nil {
		t.Fatalf("habit add: %v", err)
	}
	if err := habits.Check(st, id, now); err != nil {
		t.Fatalf("habit check: %v", err)
	}
	return st
}

func TestSummaryRendersSections(t *testing.T) {
	seedSummaryFixture(t)
	out := captureStdout(t, func() error { return runSummary([]string{"--week"}) })

	for _, want := range []string{
		"# MindForge summary",
		"## ✓ Completed (1)",
		"shipped",
		"## ⏳ Open",
		"overdue (1)",
		"late",
		"## 🍅 Focus",
		"deep work",
		"## 📓 Journal",
		"avg mood",
		"## 🌱 Habits",
		"drink water",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q\n----\n%s", want, out)
		}
	}
}

func TestSummaryEmptyState(t *testing.T) {
	withTempStore(t)
	out := captureStdout(t, func() error { return runSummary([]string{"--week"}) })
	if !strings.Contains(out, "no tasks closed in this window") {
		t.Errorf("expected empty-state hint, got:\n%s", out)
	}
}

func TestSummaryInvalidDays(t *testing.T) {
	withTempStore(t)
	if err := runSummary([]string{"--days", "0"}); err == nil {
		t.Error("expected error for non-positive --days")
	}
	if err := runSummary([]string{"--days", "abc"}); err == nil {
		t.Error("expected error for non-numeric --days")
	}
}

func TestSummaryMonthFlag(t *testing.T) {
	seedSummaryFixture(t)
	out := captureStdout(t, func() error { return runSummary([]string{"--month"}) })
	if !strings.Contains(out, "last 30 days") {
		t.Errorf("expected --month to widen window, got:\n%s", out)
	}
}
