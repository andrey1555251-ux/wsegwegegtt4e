package habits

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/andrey1555251-ux/mindforge/internal/store"
)

func newStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "data.json"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return st
}

func TestAddDuplicate(t *testing.T) {
	st := newStore(t)
	if _, err := Add(st, "drink water", 0); err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := Add(st, "Drink Water", 0); err == nil {
		t.Fatal("expected duplicate error for case-insensitive name")
	}
}

func TestCheckIsIdempotent(t *testing.T) {
	st := newStore(t)
	id, _ := Add(st, "read", 0)
	day := time.Now()
	if err := Check(st, id, day); err != nil {
		t.Fatalf("first check: %v", err)
	}
	if err := Check(st, id, day); err != nil {
		t.Fatalf("second check should be no-op: %v", err)
	}
	h, _ := Get(st, id)
	if len(h.Checks) != 1 {
		t.Fatalf("expected 1 check, got %d (%v)", len(h.Checks), h.Checks)
	}
}

func TestUncheck(t *testing.T) {
	st := newStore(t)
	id, _ := Add(st, "stretch", 0)
	day := time.Now()
	_ = Check(st, id, day)
	if err := Uncheck(st, id, day); err != nil {
		t.Fatalf("uncheck: %v", err)
	}
	h, _ := Get(st, id)
	if len(h.Checks) != 0 {
		t.Fatalf("expected empty after uncheck, got %v", h.Checks)
	}
}

func TestStreakAndBest(t *testing.T) {
	st := newStore(t)
	id, _ := Add(st, "run", 0)
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		day := now.AddDate(0, 0, -i)
		if err := Check(st, id, day); err != nil {
			t.Fatalf("check %d: %v", i, err)
		}
	}
	// Add a couple of older streaks separated by gaps
	for i := 0; i < 3; i++ {
		_ = Check(st, id, now.AddDate(0, 0, -10-i))
	}
	h, _ := Get(st, id)
	if got := Streak(h, now); got != 5 {
		t.Fatalf("Streak = %d, want 5", got)
	}
	if got := Best(h); got != 5 {
		t.Fatalf("Best = %d, want 5", got)
	}
}

func TestStreakFromYesterdayCounts(t *testing.T) {
	st := newStore(t)
	id, _ := Add(st, "meditate", 0)
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	twoDaysAgo := now.AddDate(0, 0, -2)
	_ = Check(st, id, yesterday)
	_ = Check(st, id, twoDaysAgo)
	h, _ := Get(st, id)
	if got := Streak(h, now); got != 2 {
		t.Fatalf("Streak = %d, want 2 (yesterday-counts mode)", got)
	}
}

func TestLastNDays(t *testing.T) {
	st := newStore(t)
	id, _ := Add(st, "code", 0)
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	_ = Check(st, id, now)
	_ = Check(st, id, now.AddDate(0, 0, -2))
	h, _ := Get(st, id)
	row := LastNDays(h, now, 4) // d-3, d-2, d-1, d-0
	want := []bool{false, true, false, true}
	if len(row) != len(want) {
		t.Fatalf("len = %d, want %d", len(row), len(want))
	}
	for i := range want {
		if row[i] != want[i] {
			t.Fatalf("row[%d] = %v, want %v (full %v)", i, row[i], want[i], row)
		}
	}
}

func TestRenameAndDelete(t *testing.T) {
	st := newStore(t)
	id, _ := Add(st, "old", 0)
	if err := Rename(st, id, "new"); err != nil {
		t.Fatalf("rename: %v", err)
	}
	h, _ := Get(st, id)
	if h.Name != "new" {
		t.Fatalf("rename did not stick: %q", h.Name)
	}
	if err := Delete(st, id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok := Get(st, id); ok {
		t.Fatal("habit should be gone after delete")
	}
}
