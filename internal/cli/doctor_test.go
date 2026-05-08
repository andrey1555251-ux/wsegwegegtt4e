package cli

import (
	"testing"
	"time"

	"github.com/andrey1555251-ux/mindforge/internal/store"
)

func TestAuditFindsBrokenCounters(t *testing.T) {
	now := time.Now()
	d := &store.Data{
		Tasks: []store.Task{{ID: 5, Title: "x", CreatedAt: now}},
		Notes: []store.Note{{ID: 7, Title: "n", CreatedAt: now}},
		Counters: store.Counters{
			NextNoteID:  1,
			NextTaskID:  1,
			NextHabitID: 1,
		},
	}
	issues := auditData(d)
	if len(issues) == 0 {
		t.Fatalf("expected issues, got none")
	}
	hasNote, hasTask := false, false
	for _, i := range issues {
		if contains(i, "next_note_id") {
			hasNote = true
		}
		if contains(i, "next_task_id") {
			hasTask = true
		}
	}
	if !hasNote || !hasTask {
		t.Errorf("missing counter issues: notes=%v tasks=%v", hasNote, hasTask)
	}
}

func TestRepairData(t *testing.T) {
	now := time.Now()
	doneAt := now.Add(-time.Hour)
	d := &store.Data{
		Tasks: []store.Task{
			{ID: 5, Title: "needs done_at", Done: true},                                   // missing DoneAt
			{ID: 6, Title: "stale done_at", Done: false, DoneAt: &doneAt},                  // stray DoneAt
		},
		Notes:   []store.Note{{ID: 7, Title: "n"}},
		Habits:  []store.Habit{{ID: 9, Name: "h"}},
		Journal: []store.JournalEntry{{Date: "not-a-date"}, {Date: "2026-05-01"}},
		Counters: store.Counters{
			NextNoteID:  1,
			NextTaskID:  1,
			NextHabitID: 1,
		},
	}
	repairData(d)
	if d.Counters.NextNoteID != 8 {
		t.Errorf("note counter = %d, want 8", d.Counters.NextNoteID)
	}
	if d.Counters.NextTaskID != 7 {
		t.Errorf("task counter = %d, want 7", d.Counters.NextTaskID)
	}
	if d.Counters.NextHabitID != 10 {
		t.Errorf("habit counter = %d, want 10", d.Counters.NextHabitID)
	}
	if d.Tasks[0].DoneAt == nil {
		t.Error("expected DoneAt populated for done task")
	}
	if d.Tasks[1].DoneAt != nil {
		t.Error("expected DoneAt cleared for not-done task")
	}
	if len(d.Journal) != 1 || d.Journal[0].Date != "2026-05-01" {
		t.Errorf("journal not pruned: %#v", d.Journal)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (indexOf(haystack, needle) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
