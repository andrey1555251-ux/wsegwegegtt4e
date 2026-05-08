package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/andrey1555251-ux/mindforge/internal/store"
	"github.com/andrey1555251-ux/mindforge/internal/tasks"
)

// captureStdout redirects os.Stdout while fn runs and returns whatever
// was written.  Used to test commands that print directly to stdout.
func captureStdout(t *testing.T, fn func() error) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	done := make(chan struct{})
	var buf bytes.Buffer
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	if err := fn(); err != nil {
		w.Close()
		<-done
		t.Fatalf("fn error: %v", err)
	}
	w.Close()
	<-done
	return buf.String()
}

// withTempStore points the cli.open() helper at a fresh data file in a
// new temp directory and returns the store path.
func withTempStore(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("MINDFORGE_HOME", dir)
	if _, err := store.Open(filepath.Join(dir, "data.json")); err != nil {
		t.Fatalf("seed store: %v", err)
	}
}

func TestAgendaGroupsTasksByBucket(t *testing.T) {
	withTempStore(t)
	st, err := open()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	today := now
	tomorrow := now.AddDate(0, 0, 1)
	farOff := now.AddDate(0, 0, 30)

	mustAdd := func(title string, p int, due *time.Time) {
		if _, err := tasks.Add(st, title, p, due, nil, ""); err != nil {
			t.Fatalf("add %s: %v", title, err)
		}
	}
	mustAdd("overdue task", 1, &yesterday)
	mustAdd("today task", 2, &today)
	mustAdd("tomorrow task", 3, &tomorrow)
	mustAdd("later task", 3, &farOff)
	mustAdd("no due task", 4, nil)

	out := captureStdout(t, func() error { return runAgenda([]string{"--days", "7"}) })

	for _, want := range []string{"OVERDUE", "overdue task", "today task", "tomorrow task", "Later", "later task", "no due date"} {
		if !strings.Contains(out, want) {
			t.Errorf("agenda missing %q\n%s", want, out)
		}
	}
}

func TestAgendaEmptyWindowMessage(t *testing.T) {
	withTempStore(t)
	st, err := open()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := tasks.Add(st, "no due", 3, nil, nil, ""); err != nil {
		t.Fatalf("add: %v", err)
	}
	out := captureStdout(t, func() error { return runAgenda([]string{"--days", "3"}) })
	if !strings.Contains(out, "nothing scheduled") {
		t.Errorf("expected empty-window hint, got:\n%s", out)
	}
}

func TestAgendaInvalidDaysFlag(t *testing.T) {
	withTempStore(t)
	if err := runAgenda([]string{"--days"}); err == nil {
		t.Error("expected error for missing --days value")
	}
}
