package stats

import (
	"testing"
	"time"

	"github.com/andrey1555251-ux/mindforge/internal/store"
)

func mustParse(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestComputeBasic(t *testing.T) {
	now := mustParse("2026-05-08")
	d := &store.Data{
		Notes: []store.Note{{ID: 1}, {ID: 2}},
		Tasks: []store.Task{
			{ID: 1, Title: "open"},
			{ID: 2, Title: "done", Done: true},
			{ID: 3, Title: "overdue", Due: ptr(now.AddDate(0, 0, -2))},
		},
		Pomodoros: []store.PomodoroLog{
			{StartedAt: now, FinishedAt: now.Add(25 * time.Minute), Minutes: 25},
			{StartedAt: now.AddDate(0, 0, -1), FinishedAt: now.AddDate(0, 0, -1).Add(25 * time.Minute), Minutes: 25},
		},
		Journal: []store.JournalEntry{
			{Date: "2026-05-08", Sections: []store.Section{{Body: "a"}}},
			{Date: "2026-05-07", Sections: []store.Section{{Body: "b"}}},
			{Date: "2026-05-06", Sections: []store.Section{{Body: "c"}}},
		},
	}
	s := Compute(d, now)
	if s.Notes != 2 {
		t.Errorf("Notes = %d, want 2", s.Notes)
	}
	if s.OpenTasks != 2 {
		t.Errorf("OpenTasks = %d, want 2", s.OpenTasks)
	}
	if s.Tasks != 3 {
		t.Errorf("Tasks = %d, want 3", s.Tasks)
	}
	if s.Overdue != 1 {
		t.Errorf("Overdue = %d, want 1", s.Overdue)
	}
	if s.PomodorosToday != 1 {
		t.Errorf("PomodorosToday = %d, want 1", s.PomodorosToday)
	}
	if s.PomodorosTotal != 2 {
		t.Errorf("PomodorosTotal = %d, want 2", s.PomodorosTotal)
	}
	if s.FocusMinutesToday != 25 {
		t.Errorf("FocusMinutesToday = %d, want 25", s.FocusMinutesToday)
	}
	if s.JournalDays != 3 {
		t.Errorf("JournalDays = %d, want 3", s.JournalDays)
	}
	if s.JournalStreak != 3 {
		t.Errorf("JournalStreak = %d, want 3", s.JournalStreak)
	}
}

func TestTopTags(t *testing.T) {
	d := &store.Data{
		Notes: []store.Note{
			{Tags: []string{"work", "ideas"}},
			{Tags: []string{"work"}},
		},
		Tasks: []store.Task{
			{Tags: []string{"work", "urgent"}},
			{Tags: []string{"home"}},
		},
	}
	got := TopTags(d, 5)
	// expect work=3, ideas=1, urgent=1, home=1
	wantTop := "work"
	if len(got) == 0 || got[0].Tag != wantTop {
		t.Fatalf("top tag = %v, want first to be %q", got, wantTop)
	}
	if got[0].Count != 3 {
		t.Fatalf("top tag count = %d, want 3", got[0].Count)
	}
}

func ptr(t time.Time) *time.Time { return &t }
