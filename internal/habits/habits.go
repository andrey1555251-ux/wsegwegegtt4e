// Package habits implements a tiny daily-habit tracker.  A habit is a
// named recurring intention (e.g. "drink water", "10 push-ups") and
// each day the user can mark it as done.  We compute streaks the same
// way the journal does so it feels consistent.
package habits

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/andrey1555251-ux/mindforge/internal/store"
)

const dateFmt = "2006-01-02"

// Add creates a new habit and returns its id.  Names are unique
// (case-insensitive) to avoid accidental dupes.
func Add(st *store.Store, name string, target int) (int, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, errors.New("name is required")
	}
	if target <= 0 {
		target = 1
	}
	var id int
	err := st.Use(func(d *store.Data) error {
		for _, h := range d.Habits {
			if strings.EqualFold(h.Name, name) {
				return fmt.Errorf("habit %q already exists (#%d)", h.Name, h.ID)
			}
		}
		if d.Counters.NextHabitID == 0 {
			d.Counters.NextHabitID = 1
		}
		id = d.Counters.NextHabitID
		d.Counters.NextHabitID++
		d.Habits = append(d.Habits, store.Habit{
			ID:        id,
			Name:      name,
			Target:    target,
			CreatedAt: time.Now(),
		})
		return nil
	})
	return id, err
}

// Delete removes a habit and its check-ins.
func Delete(st *store.Store, id int) error {
	return st.Use(func(d *store.Data) error {
		out := d.Habits[:0]
		found := false
		for _, h := range d.Habits {
			if h.ID == id {
				found = true
				continue
			}
			out = append(out, h)
		}
		if !found {
			return fmt.Errorf("habit %d not found", id)
		}
		d.Habits = out
		return nil
	})
}

// Rename changes the display name of an existing habit.
func Rename(st *store.Store, id int, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("name is required")
	}
	return st.Use(func(d *store.Data) error {
		for i := range d.Habits {
			if d.Habits[i].ID == id {
				d.Habits[i].Name = name
				return nil
			}
		}
		return fmt.Errorf("habit %d not found", id)
	})
}

// Check records a completion for the given day (default: today).  If
// the habit is already checked for that day we return nil — calling
// `check` repeatedly is a no-op so scripts can stay simple.
func Check(st *store.Store, id int, day time.Time) error {
	d := day.Format(dateFmt)
	return st.Use(func(data *store.Data) error {
		var hab *store.Habit
		for i := range data.Habits {
			if data.Habits[i].ID == id {
				hab = &data.Habits[i]
				break
			}
		}
		if hab == nil {
			return fmt.Errorf("habit %d not found", id)
		}
		for _, c := range hab.Checks {
			if c == d {
				return nil
			}
		}
		hab.Checks = append(hab.Checks, d)
		sort.Strings(hab.Checks)
		return nil
	})
}

// Uncheck removes a completion for the given day.
func Uncheck(st *store.Store, id int, day time.Time) error {
	d := day.Format(dateFmt)
	return st.Use(func(data *store.Data) error {
		for i := range data.Habits {
			if data.Habits[i].ID != id {
				continue
			}
			out := data.Habits[i].Checks[:0]
			for _, c := range data.Habits[i].Checks {
				if c != d {
					out = append(out, c)
				}
			}
			data.Habits[i].Checks = out
			return nil
		}
		return fmt.Errorf("habit %d not found", id)
	})
}

// All returns a sorted copy of all habits.
func All(st *store.Store) []store.Habit {
	d, _ := st.Snapshot()
	if d == nil {
		return nil
	}
	out := make([]store.Habit, len(d.Habits))
	copy(out, d.Habits)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Get returns one habit by id.
func Get(st *store.Store, id int) (store.Habit, bool) {
	d, _ := st.Snapshot()
	if d == nil {
		return store.Habit{}, false
	}
	for _, h := range d.Habits {
		if h.ID == id {
			return h, true
		}
	}
	return store.Habit{}, false
}

// Streak returns the number of consecutive days ending at `today` that
// the habit was checked.  Today itself counting is optional — if the
// habit hasn't been checked today we still count yesterday's streak so
// the user is not punished for not having ticked the box yet.
func Streak(h store.Habit, today time.Time) int {
	if len(h.Checks) == 0 {
		return 0
	}
	set := make(map[string]bool, len(h.Checks))
	for _, c := range h.Checks {
		set[c] = true
	}
	day := startOfDay(today)
	if !set[day.Format(dateFmt)] {
		// allow yesterday-counts behaviour
		day = day.AddDate(0, 0, -1)
		if !set[day.Format(dateFmt)] {
			return 0
		}
	}
	streak := 0
	for set[day.Format(dateFmt)] {
		streak++
		day = day.AddDate(0, 0, -1)
	}
	return streak
}

// Best returns the longest streak the habit has ever achieved.
func Best(h store.Habit) int {
	if len(h.Checks) == 0 {
		return 0
	}
	dates := make([]time.Time, 0, len(h.Checks))
	for _, c := range h.Checks {
		t, err := time.Parse(dateFmt, c)
		if err != nil {
			continue
		}
		dates = append(dates, t)
	}
	if len(dates) == 0 {
		return 0
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	best, cur := 1, 1
	for i := 1; i < len(dates); i++ {
		if dates[i].Sub(dates[i-1]) == 24*time.Hour {
			cur++
		} else if !sameDay(dates[i], dates[i-1]) {
			cur = 1
		}
		if cur > best {
			best = cur
		}
	}
	return best
}

// CheckedOn returns true if the habit was completed on the given day.
func CheckedOn(h store.Habit, day time.Time) bool {
	target := day.Format(dateFmt)
	for _, c := range h.Checks {
		if c == target {
			return true
		}
	}
	return false
}

// LastNDays returns a slice of bools for the last N days, oldest-first,
// indicating whether the habit was checked on that day.
func LastNDays(h store.Habit, end time.Time, n int) []bool {
	out := make([]bool, n)
	set := make(map[string]bool, len(h.Checks))
	for _, c := range h.Checks {
		set[c] = true
	}
	for i := 0; i < n; i++ {
		day := startOfDay(end).AddDate(0, 0, -(n - 1 - i))
		out[i] = set[day.Format(dateFmt)]
	}
	return out
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}
