// Package tasks manages the user's TODO list.
package tasks

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/andrey1555251-ux/mindforge/internal/store"
)

// Add inserts a new task and returns its id.
func Add(s *store.Store, title string, priority int, due *time.Time, tags []string, notes string) (int, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return 0, errors.New("title is required")
	}
	if priority < 1 || priority > 5 {
		priority = 3
	}
	tags = normalizeTags(tags)

	var id int
	err := s.Use(func(d *store.Data) error {
		id = d.Counters.NextTaskID
		d.Counters.NextTaskID++
		d.Tasks = append(d.Tasks, store.Task{
			ID:        id,
			Title:     title,
			Notes:     notes,
			Priority:  priority,
			Due:       due,
			Tags:      tags,
			CreatedAt: time.Now(),
		})
		return nil
	})
	return id, err
}

// Done flips done=true on a task and stores DoneAt.
func Done(s *store.Store, id int) error {
	return s.Use(func(d *store.Data) error {
		for i := range d.Tasks {
			if d.Tasks[i].ID != id {
				continue
			}
			now := time.Now()
			d.Tasks[i].Done = true
			d.Tasks[i].DoneAt = &now
			return nil
		}
		return fmt.Errorf("task %d not found", id)
	})
}

// Reopen marks a previously completed task as not done.
func Reopen(s *store.Store, id int) error {
	return s.Use(func(d *store.Data) error {
		for i := range d.Tasks {
			if d.Tasks[i].ID != id {
				continue
			}
			d.Tasks[i].Done = false
			d.Tasks[i].DoneAt = nil
			return nil
		}
		return fmt.Errorf("task %d not found", id)
	})
}

// Delete removes a task.
func Delete(s *store.Store, id int) error {
	return s.Use(func(d *store.Data) error {
		for i := range d.Tasks {
			if d.Tasks[i].ID == id {
				d.Tasks = append(d.Tasks[:i], d.Tasks[i+1:]...)
				return nil
			}
		}
		return fmt.Errorf("task %d not found", id)
	})
}

// Update edits selected fields of a task in place.
func Update(s *store.Store, id int, title, notes *string, priority *int, due *time.Time, clearDue bool, tags []string, replaceTags bool) error {
	return s.Use(func(d *store.Data) error {
		for i := range d.Tasks {
			if d.Tasks[i].ID != id {
				continue
			}
			if title != nil {
				d.Tasks[i].Title = *title
			}
			if notes != nil {
				d.Tasks[i].Notes = *notes
			}
			if priority != nil {
				p := *priority
				if p < 1 {
					p = 1
				}
				if p > 5 {
					p = 5
				}
				d.Tasks[i].Priority = p
			}
			if clearDue {
				d.Tasks[i].Due = nil
			} else if due != nil {
				due := *due
				d.Tasks[i].Due = &due
			}
			if replaceTags {
				d.Tasks[i].Tags = normalizeTags(tags)
			} else if len(tags) > 0 {
				d.Tasks[i].Tags = mergeTags(d.Tasks[i].Tags, tags)
			}
			return nil
		}
		return fmt.Errorf("task %d not found", id)
	})
}

// Get returns a copy of a task.
func Get(s *store.Store, id int) (store.Task, bool) {
	var t store.Task
	var ok bool
	s.View(func(d *store.Data) {
		for _, x := range d.Tasks {
			if x.ID == id {
				t = x
				ok = true
				return
			}
		}
	})
	return t, ok
}

// FilterOpts controls List below.
type FilterOpts struct {
	IncludeDone bool
	OnlyDone    bool
	Tag         string
	Query       string
	OverdueOnly bool
	DueSoonHrs  int // 0 = ignore
}

// List returns tasks matching opts, sorted by smart key.
func List(s *store.Store, opts FilterOpts) []store.Task {
	q := strings.ToLower(strings.TrimSpace(opts.Query))
	tag := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(opts.Tag, "#")))

	var out []store.Task
	now := time.Now()
	s.View(func(d *store.Data) {
		for _, t := range d.Tasks {
			if !opts.IncludeDone && !opts.OnlyDone && t.Done {
				continue
			}
			if opts.OnlyDone && !t.Done {
				continue
			}
			if tag != "" && !containsString(t.Tags, tag) {
				continue
			}
			if q != "" {
				hay := strings.ToLower(t.Title + " " + t.Notes)
				if !strings.Contains(hay, q) {
					continue
				}
			}
			if opts.OverdueOnly {
				if t.Done || t.Due == nil || !t.Due.Before(now) {
					continue
				}
			}
			if opts.DueSoonHrs > 0 {
				if t.Done || t.Due == nil {
					continue
				}
				diff := t.Due.Sub(now)
				if diff < 0 || diff > time.Duration(opts.DueSoonHrs)*time.Hour {
					continue
				}
			}
			out = append(out, t)
		}
	})

	sort.SliceStable(out, func(i, j int) bool {
		// Done tasks always sink to the bottom.
		if out[i].Done != out[j].Done {
			return !out[i].Done
		}
		// Then by overdueness/due date proximity.
		di, dj := dueKey(out[i], now), dueKey(out[j], now)
		if di != dj {
			return di < dj
		}
		// Then by priority (smaller is higher).
		if out[i].Priority != out[j].Priority {
			return out[i].Priority < out[j].Priority
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func dueKey(t store.Task, now time.Time) int64 {
	if t.Due == nil {
		// no due date — rank just below tasks due in 30 days.
		return now.Add(30 * 24 * time.Hour).Unix()
	}
	return t.Due.Unix()
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func normalizeTags(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, t := range in {
		t = strings.ToLower(strings.TrimSpace(t))
		t = strings.TrimPrefix(t, "#")
		if t == "" {
			continue
		}
		if _, dup := seen[t]; dup {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

func mergeTags(existing, additions []string) []string {
	return normalizeTags(append(append([]string{}, existing...), additions...))
}
