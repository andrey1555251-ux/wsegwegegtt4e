package journal

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/andrey1555251-ux/mindforge/internal/store"
)

func newStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "data.json"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	return s
}

func TestAppendCreatesEntry(t *testing.T) {
	s := newStore(t)
	day := time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC)
	if err := Append(s, day, "first"); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := Append(s, day, "second"); err != nil {
		t.Fatalf("append 2: %v", err)
	}
	e, ok := Get(s, day)
	if !ok {
		t.Fatalf("entry missing")
	}
	if len(e.Sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(e.Sections))
	}
	if e.Sections[0].Body != "first" || e.Sections[1].Body != "second" {
		t.Fatalf("bodies wrong: %+v", e.Sections)
	}
}

func TestSetMood(t *testing.T) {
	s := newStore(t)
	day := time.Now()
	if err := SetMood(s, day, 4); err != nil {
		t.Fatalf("set mood: %v", err)
	}
	e, _ := Get(s, day)
	if e.Mood != 4 {
		t.Fatalf("mood not set: %d", e.Mood)
	}
	if err := SetMood(s, day, 7); err == nil {
		t.Fatalf("expected error for invalid mood")
	}
}

func TestStreak(t *testing.T) {
	s := newStore(t)
	now := time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		_ = Append(s, now.AddDate(0, 0, -i), "x")
	}
	if got := Streak(s, now); got != 3 {
		t.Fatalf("expected streak 3, got %d", got)
	}
	// Add a gap.
	_ = Append(s, now.AddDate(0, 0, -5), "x")
	if got := Streak(s, now); got != 3 {
		t.Fatalf("gap should not change streak, got %d", got)
	}
}
