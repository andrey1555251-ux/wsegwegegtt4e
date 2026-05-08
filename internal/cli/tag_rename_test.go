package cli

import (
	"strings"
	"testing"

	"github.com/andrey1555251-ux/mindforge/internal/notes"
	"github.com/andrey1555251-ux/mindforge/internal/tasks"
)

// reopenSnapshot returns a fresh Data from disk via a new store
// instance — necessary because cli.runX functions open their own
// store, so any in-test handle becomes stale after a write.
func reopenSnapshot(t *testing.T) (notesTags, taskTags []string) {
	t.Helper()
	st, err := open()
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	d, err := st.Snapshot()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if len(d.Notes) > 0 {
		notesTags = d.Notes[0].Tags
	}
	if len(d.Tasks) > 0 {
		taskTags = d.Tasks[0].Tags
	}
	return
}

func TestTagRenameAcrossNotesAndTasks(t *testing.T) {
	withTempStore(t)
	st, err := open()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := notes.Add(st, "n1", "body", []string{"work", "home"}); err != nil {
		t.Fatalf("note add: %v", err)
	}
	if _, err := tasks.Add(st, "t1", 3, nil, []string{"work"}, ""); err != nil {
		t.Fatalf("task add: %v", err)
	}

	out := captureStdout(t, func() error { return runTagRename([]string{"work", "job"}) })
	if !strings.Contains(out, "renamed #work → #job") {
		t.Errorf("missing rename feedback:\n%s", out)
	}

	noteTags, taskTags := reopenSnapshot(t)
	if joined := strings.Join(noteTags, " "); !strings.Contains(joined, "job") || strings.Contains(joined, "work") {
		t.Errorf("note tags %v not updated", noteTags)
	}
	if joined := strings.Join(taskTags, " "); !strings.Contains(joined, "job") || strings.Contains(joined, "work") {
		t.Errorf("task tags %v not updated", taskTags)
	}
}

func TestTagRenameDeleteWhenToOmitted(t *testing.T) {
	withTempStore(t)
	st, err := open()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := tasks.Add(st, "t1", 3, nil, []string{"a", "b"}, ""); err != nil {
		t.Fatalf("add: %v", err)
	}
	out := captureStdout(t, func() error { return runTagRename([]string{"a"}) })
	if !strings.Contains(out, "removed #a") {
		t.Errorf("expected delete message, got:\n%s", out)
	}
	_, taskTags := reopenSnapshot(t)
	if got := strings.Join(taskTags, " "); strings.Contains(got, "a") && !strings.Contains(got, "b") {
		t.Errorf("tag a not removed: %q", got)
	}
}

func TestTagRenameStripsLeadingHash(t *testing.T) {
	withTempStore(t)
	st, _ := open()
	if _, err := tasks.Add(st, "t1", 3, nil, []string{"work"}, ""); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := runTagRename([]string{"#work", "#job"}); err != nil {
		t.Fatalf("rename: %v", err)
	}
	_, taskTags := reopenSnapshot(t)
	if got := strings.Join(taskTags, " "); !strings.Contains(got, "job") {
		t.Errorf("tag not renamed (got %q)", got)
	}
}

func TestTagRenameNoOpWhenMissing(t *testing.T) {
	withTempStore(t)
	out := captureStdout(t, func() error { return runTagRename([]string{"ghost", "phantom"}) })
	if !strings.Contains(out, "no items had #ghost") {
		t.Errorf("expected no-op hint, got:\n%s", out)
	}
}

func TestTagRenameUsage(t *testing.T) {
	withTempStore(t)
	if err := runTagRename(nil); err == nil {
		t.Error("expected usage error with no args")
	}
	if err := runTagRename([]string{"", "x"}); err == nil {
		t.Error("expected error for empty source tag")
	}
}
