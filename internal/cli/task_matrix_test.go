package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/andrey1555251-ux/mindforge/internal/tasks"
)

func TestTaskMatrixGroupsAllFourQuadrants(t *testing.T) {
	withTempStore(t)
	st, err := open()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	soon := time.Now().Add(2 * time.Hour)
	later := time.Now().Add(10 * 24 * time.Hour)
	if _, err := tasks.Add(st, "ship", 1, &soon, nil, ""); err != nil {
		t.Fatalf("ship: %v", err)
	}
	if _, err := tasks.Add(st, "plan", 1, &later, nil, ""); err != nil {
		t.Fatalf("plan: %v", err)
	}
	if _, err := tasks.Add(st, "errand", 5, &soon, nil, ""); err != nil {
		t.Fatalf("errand: %v", err)
	}
	if _, err := tasks.Add(st, "scroll", 5, nil, nil, ""); err != nil {
		t.Fatalf("scroll: %v", err)
	}

	out := captureStdout(t, func() error { return runTaskMatrix(nil) })
	for _, want := range []string{"DO", "PLAN", "DELEGATE", "DROP", "ship", "plan", "errand", "scroll"} {
		if !strings.Contains(out, want) {
			t.Errorf("matrix missing %q in:\n%s", want, out)
		}
	}
}

func TestTaskMatrixEmptyShowsDashes(t *testing.T) {
	withTempStore(t)
	out := captureStdout(t, func() error { return runTaskMatrix(nil) })
	if strings.Count(out, "—") < 4 {
		t.Errorf("expected 4 empty-quadrant dashes, got:\n%s", out)
	}
}
