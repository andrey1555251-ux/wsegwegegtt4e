package tasks

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

func TestTaskLifecycle(t *testing.T) {
	s := newStore(t)
	due := time.Now().Add(24 * time.Hour)
	id, err := Add(s, "ship release", 1, &due, []string{"work"}, "subtasks: build, test")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if id == 0 {
		t.Fatalf("zero id")
	}

	if err := Done(s, id); err != nil {
		t.Fatalf("done: %v", err)
	}
	got, _ := Get(s, id)
	if !got.Done {
		t.Fatalf("not done")
	}
	if got.DoneAt == nil {
		t.Fatalf("done_at not set")
	}

	if err := Reopen(s, id); err != nil {
		t.Fatalf("reopen: %v", err)
	}
	got, _ = Get(s, id)
	if got.Done {
		t.Fatalf("still done after reopen")
	}
	if got.DoneAt != nil {
		t.Fatalf("done_at not cleared")
	}
}

func TestTaskListSorts(t *testing.T) {
	s := newStore(t)
	now := time.Now()
	soon := now.Add(2 * time.Hour)
	later := now.Add(72 * time.Hour)
	overdue := now.Add(-1 * time.Hour)

	a, _ := Add(s, "a future low pri", 5, &later, nil, "")
	b, _ := Add(s, "b due soon high pri", 1, &soon, nil, "")
	c, _ := Add(s, "c overdue mid pri", 3, &overdue, nil, "")

	out := List(s, FilterOpts{})
	if len(out) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(out))
	}
	if out[0].ID != c {
		t.Fatalf("expected overdue first, got %d", out[0].ID)
	}
	if out[1].ID != b {
		t.Fatalf("expected due-soon second, got %d", out[1].ID)
	}
	if out[2].ID != a {
		t.Fatalf("expected later third, got %d", out[2].ID)
	}
}

func TestTaskListFilters(t *testing.T) {
	s := newStore(t)
	due := time.Now().Add(24 * time.Hour)
	overdue := time.Now().Add(-time.Hour)
	id1, _ := Add(s, "buy bread", 3, &due, []string{"home"}, "")
	id2, _ := Add(s, "pay bills", 1, &overdue, []string{"home", "money"}, "")
	_, _ = Add(s, "ship release", 1, &due, []string{"work"}, "")

	out := List(s, FilterOpts{Tag: "home"})
	if len(out) != 2 {
		t.Fatalf("expected 2 home tasks, got %d", len(out))
	}

	out = List(s, FilterOpts{OverdueOnly: true})
	if len(out) != 1 || out[0].ID != id2 {
		t.Fatalf("overdue filter: %+v", out)
	}

	out = List(s, FilterOpts{Query: "bread"})
	if len(out) != 1 || out[0].ID != id1 {
		t.Fatalf("query filter: %+v", out)
	}

	if err := Done(s, id1); err != nil {
		t.Fatalf("done: %v", err)
	}
	out = List(s, FilterOpts{})
	if len(out) != 2 {
		t.Fatalf("done task should be hidden by default")
	}
	out = List(s, FilterOpts{IncludeDone: true})
	if len(out) != 3 {
		t.Fatalf("expected all with IncludeDone")
	}
	out = List(s, FilterOpts{OnlyDone: true})
	if len(out) != 1 {
		t.Fatalf("expected only done")
	}
}

func TestTaskUpdate(t *testing.T) {
	s := newStore(t)
	id, _ := Add(s, "title", 3, nil, nil, "")
	newTitle := "renamed"
	if err := Update(s, id, &newTitle, nil, nil, nil, false, []string{"foo"}, false); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ := Get(s, id)
	if got.Title != newTitle {
		t.Fatalf("title not changed")
	}
	if len(got.Tags) != 1 || got.Tags[0] != "foo" {
		t.Fatalf("tags wrong: %+v", got.Tags)
	}

	clear := true
	due := time.Now().Add(time.Hour)
	if err := Update(s, id, nil, nil, nil, &due, false, nil, false); err != nil {
		t.Fatalf("set due: %v", err)
	}
	got, _ = Get(s, id)
	if got.Due == nil {
		t.Fatalf("due not set")
	}
	if err := Update(s, id, nil, nil, nil, nil, clear, nil, false); err != nil {
		t.Fatalf("clear due: %v", err)
	}
	got, _ = Get(s, id)
	if got.Due != nil {
		t.Fatalf("due not cleared")
	}
}
