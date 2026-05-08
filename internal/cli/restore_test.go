package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andrey1555251-ux/mindforge/internal/tasks"
)

func TestRestoreReplacesDataAndBacksUp(t *testing.T) {
	withTempStore(t)
	st, err := open()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := tasks.Add(st, "v1", 3, nil, nil, ""); err != nil {
		t.Fatalf("add v1: %v", err)
	}
	// Snapshot the v1 state to use later as the restore source.
	snap := filepath.Join(t.TempDir(), "snap.json")
	raw, err := os.ReadFile(st.Path())
	if err != nil {
		t.Fatalf("read store: %v", err)
	}
	if err := os.WriteFile(snap, raw, 0o644); err != nil {
		t.Fatalf("write snap: %v", err)
	}
	// Mutate the store: add a second task, so v2 != v1.
	st2, err := open()
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if _, err := tasks.Add(st2, "v2", 3, nil, nil, ""); err != nil {
		t.Fatalf("add v2: %v", err)
	}

	out := captureStdout(t, func() error { return runRestore([]string{snap}) })
	if !strings.Contains(out, "restored:") || !strings.Contains(out, "previous state saved to:") {
		t.Errorf("missing restore output:\n%s", out)
	}

	// After restore we should only have one task again.
	st3, err := open()
	if err != nil {
		t.Fatalf("reopen after restore: %v", err)
	}
	got := tasks.List(st3, tasks.FilterOpts{})
	if len(got) != 1 || got[0].Title != "v1" {
		t.Errorf("post-restore tasks = %#v, want only v1", got)
	}
}

func TestRestoreRejectsGarbage(t *testing.T) {
	withTempStore(t)
	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte("{not json"), 0o644); err != nil {
		t.Fatalf("write bad: %v", err)
	}
	if err := runRestore([]string{bad}); err == nil {
		t.Error("expected error for invalid backup")
	}
}

func TestRestoreUsage(t *testing.T) {
	withTempStore(t)
	if err := runRestore(nil); err == nil {
		t.Error("expected usage error with no args")
	}
}
