package notes

import (
	"path/filepath"
	"testing"

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

func TestAddAndFilter(t *testing.T) {
	s := newStore(t)
	id1, err := Add(s, "shopping list", "milk, eggs", []string{"home", "errand"})
	if err != nil {
		t.Fatalf("add 1: %v", err)
	}
	id2, err := Add(s, "ship release", "v0.1 plan", []string{"work", "release"})
	if err != nil {
		t.Fatalf("add 2: %v", err)
	}
	if id1 == id2 {
		t.Fatalf("ids reused: %d", id1)
	}

	out := Filter(s, FilterOpts{Tags: []string{"work"}})
	if len(out) != 1 || out[0].ID != id2 {
		t.Fatalf("tag filter mismatch: %+v", out)
	}

	out = Filter(s, FilterOpts{Query: "MILK", IncludeBody: true})
	if len(out) != 1 || out[0].ID != id1 {
		t.Fatalf("body search failed: %+v", out)
	}

	if err := Pin(s, id2, true); err != nil {
		t.Fatalf("pin: %v", err)
	}
	out = Filter(s, FilterOpts{SortByPinned: true})
	if len(out) != 2 || out[0].ID != id2 {
		t.Fatalf("pin sort failed: %+v", out)
	}
}

func TestUpdateNote(t *testing.T) {
	s := newStore(t)
	id, _ := Add(s, "old title", "old body", []string{"x"})
	newTitle := "new title"
	newBody := "new body"
	if err := Update(s, id, &newTitle, &newBody, []string{"y"}, false); err != nil {
		t.Fatalf("update: %v", err)
	}
	n, ok := Get(s, id)
	if !ok {
		t.Fatalf("note vanished")
	}
	if n.Title != newTitle || n.Body != newBody {
		t.Fatalf("update did not stick: %+v", n)
	}
	if len(n.Tags) != 2 || n.Tags[0] != "x" || n.Tags[1] != "y" {
		t.Fatalf("tags merge failed: %+v", n.Tags)
	}

	if err := Update(s, id, nil, nil, []string{"only"}, true); err != nil {
		t.Fatalf("replace: %v", err)
	}
	n, _ = Get(s, id)
	if len(n.Tags) != 1 || n.Tags[0] != "only" {
		t.Fatalf("tags replace failed: %+v", n.Tags)
	}
}

func TestDeleteNote(t *testing.T) {
	s := newStore(t)
	id, _ := Add(s, "x", "y", nil)
	if err := Delete(s, id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok := Get(s, id); ok {
		t.Fatalf("note still present after delete")
	}
}

func TestNormaliseTags(t *testing.T) {
	got := normalizeTags([]string{"#Foo", " bar ", "FOO", ""})
	want := []string{"bar", "foo"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}
