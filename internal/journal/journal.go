// Package journal manages daily reflections.
package journal

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/andrey1555251-ux/mindforge/internal/store"
)

// Append adds a section of text to today's entry (or the entry for the
// given date when not zero).  An entry is created on first append.
func Append(s *store.Store, date time.Time, body string) error {
	body = strings.TrimSpace(body)
	if body == "" {
		return errors.New("body is empty")
	}
	key := date.Format("2006-01-02")
	return s.Use(func(d *store.Data) error {
		for i := range d.Journal {
			if d.Journal[i].Date == key {
				d.Journal[i].Sections = append(d.Journal[i].Sections, store.Section{At: time.Now(), Body: body})
				return nil
			}
		}
		d.Journal = append(d.Journal, store.JournalEntry{
			Date:     key,
			Sections: []store.Section{{At: time.Now(), Body: body}},
		})
		return nil
	})
}

// SetMood records a mood (1..5) for the given date.
func SetMood(s *store.Store, date time.Time, mood int) error {
	if mood < 1 || mood > 5 {
		return errors.New("mood must be between 1 and 5")
	}
	key := date.Format("2006-01-02")
	return s.Use(func(d *store.Data) error {
		for i := range d.Journal {
			if d.Journal[i].Date == key {
				d.Journal[i].Mood = mood
				return nil
			}
		}
		d.Journal = append(d.Journal, store.JournalEntry{
			Date: key,
			Mood: mood,
		})
		return nil
	})
}

// Get returns the entry for date if any.
func Get(s *store.Store, date time.Time) (store.JournalEntry, bool) {
	key := date.Format("2006-01-02")
	var found store.JournalEntry
	var ok bool
	s.View(func(d *store.Data) {
		for _, e := range d.Journal {
			if e.Date == key {
				found = e
				ok = true
				return
			}
		}
	})
	return found, ok
}

// All returns the journal sorted from newest to oldest.
func All(s *store.Store) []store.JournalEntry {
	var out []store.JournalEntry
	s.View(func(d *store.Data) {
		out = append(out, d.Journal...)
	})
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Date > out[j].Date
	})
	return out
}

// Streak returns the count of consecutive days up to today that have
// at least one journal section.
func Streak(s *store.Store, today time.Time) int {
	dates := map[string]struct{}{}
	s.View(func(d *store.Data) {
		for _, e := range d.Journal {
			if len(e.Sections) > 0 {
				dates[e.Date] = struct{}{}
			}
		}
	})
	count := 0
	for cur := today; ; cur = cur.AddDate(0, 0, -1) {
		key := cur.Format("2006-01-02")
		if _, ok := dates[key]; !ok {
			break
		}
		count++
	}
	return count
}
