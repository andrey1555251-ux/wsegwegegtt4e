// Package stats summarises what's in the store: counts, streaks,
// pomodoro time per day, etc.
package stats

import (
	"sort"
	"time"

	"github.com/andrey1555251-ux/mindforge/internal/store"
)

// Summary is the overall snapshot shown by `mindforge stats`.
type Summary struct {
	Notes              int
	Tasks              int
	OpenTasks          int
	Overdue            int
	JournalDays        int
	JournalStreak      int
	PomodorosToday     int
	PomodorosThisWeek  int
	PomodorosTotal     int
	FocusMinutesToday  int
	FocusMinutesTotal  int
	HeatLastWeeks      [7][7]int // 7 weeks x 7 weekdays of pomodoro counts
}

// Compute walks the data and returns a populated summary.
func Compute(d *store.Data, now time.Time) Summary {
	s := Summary{Notes: len(d.Notes), Tasks: len(d.Tasks)}
	for _, t := range d.Tasks {
		if !t.Done {
			s.OpenTasks++
			if t.Due != nil && t.Due.Before(now) {
				s.Overdue++
			}
		}
	}
	dates := map[string]struct{}{}
	for _, j := range d.Journal {
		if len(j.Sections) > 0 {
			dates[j.Date] = struct{}{}
		}
	}
	s.JournalDays = len(dates)
	for cur := now; ; cur = cur.AddDate(0, 0, -1) {
		key := cur.Format("2006-01-02")
		if _, ok := dates[key]; !ok {
			break
		}
		s.JournalStreak++
	}

	weekStart := startOfWeek(now)
	today := startOfDay(now)
	for _, p := range d.Pomodoros {
		if p.Interrupted {
			continue
		}
		s.PomodorosTotal++
		s.FocusMinutesTotal += p.Minutes
		if !p.StartedAt.Before(today) && p.StartedAt.Before(today.AddDate(0, 0, 1)) {
			s.PomodorosToday++
			s.FocusMinutesToday += p.Minutes
		}
		if !p.StartedAt.Before(weekStart) {
			s.PomodorosThisWeek++
		}

		// Heat-map cell for the last 7 weeks.
		dayDelta := int(today.Sub(startOfDay(p.StartedAt)).Hours() / 24)
		if dayDelta >= 0 && dayDelta < 49 {
			weekIdx := dayDelta / 7
			weekday := int(p.StartedAt.Weekday())
			s.HeatLastWeeks[weekIdx][weekday]++
		}
	}
	return s
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func startOfWeek(t time.Time) time.Time {
	d := startOfDay(t)
	weekday := int(d.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return d.AddDate(0, 0, -(weekday - 1))
}

// FocusByLabel groups completed pomodoros by their label and totals
// the focus minutes spent on each.  An empty label is shown as
// "(unlabeled)".  Sorted by minutes descending.
func FocusByLabel(d *store.Data, since time.Time, n int) []FocusRow {
	totals := map[string]int{}
	rounds := map[string]int{}
	for _, p := range d.Pomodoros {
		if p.Interrupted {
			continue
		}
		if !p.StartedAt.After(since) && !since.IsZero() {
			continue
		}
		key := p.Label
		if key == "" {
			key = "(unlabeled)"
		}
		totals[key] += p.Minutes
		rounds[key]++
	}
	out := make([]FocusRow, 0, len(totals))
	for k, v := range totals {
		out = append(out, FocusRow{Label: k, Minutes: v, Rounds: rounds[k]})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Minutes != out[j].Minutes {
			return out[i].Minutes > out[j].Minutes
		}
		return out[i].Label < out[j].Label
	})
	if n > 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// FocusRow is one row of FocusByLabel.
type FocusRow struct {
	Label   string
	Minutes int
	Rounds  int
}

// TopTags returns the most common tags across notes and tasks.
func TopTags(d *store.Data, n int) []TagCount {
	counts := map[string]int{}
	for _, x := range d.Notes {
		for _, t := range x.Tags {
			counts[t]++
		}
	}
	for _, x := range d.Tasks {
		for _, t := range x.Tags {
			counts[t]++
		}
	}
	out := make([]TagCount, 0, len(counts))
	for k, v := range counts {
		out = append(out, TagCount{Tag: k, Count: v})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Tag < out[j].Tag
	})
	if n > 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// TagCount is a tag usage row.
type TagCount struct {
	Tag   string
	Count int
}
