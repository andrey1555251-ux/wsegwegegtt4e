// Package notes manages the user's notes inside the store.
package notes

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/andrey1555251-ux/mindforge/internal/store"
)

// Add creates a new note and returns its id.
func Add(s *store.Store, title, body string, tags []string) (int, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return 0, errors.New("title is required")
	}
	tags = normalizeTags(tags)

	var id int
	err := s.Use(func(d *store.Data) error {
		id = d.Counters.NextNoteID
		d.Counters.NextNoteID++
		now := time.Now()
		d.Notes = append(d.Notes, store.Note{
			ID:        id,
			Title:     title,
			Body:      body,
			Tags:      tags,
			CreatedAt: now,
			UpdatedAt: now,
		})
		return nil
	})
	return id, err
}

// Update replaces title/body/tags of an existing note.  Empty strings
// keep the previous value so callers can update one field at a time.
func Update(s *store.Store, id int, title, body *string, tags []string, replaceTags bool) error {
	return s.Use(func(d *store.Data) error {
		for i := range d.Notes {
			if d.Notes[i].ID != id {
				continue
			}
			if title != nil {
				d.Notes[i].Title = *title
			}
			if body != nil {
				d.Notes[i].Body = *body
			}
			if replaceTags {
				d.Notes[i].Tags = normalizeTags(tags)
			} else if len(tags) > 0 {
				d.Notes[i].Tags = mergeTags(d.Notes[i].Tags, tags)
			}
			d.Notes[i].UpdatedAt = time.Now()
			return nil
		}
		return fmt.Errorf("note %d not found", id)
	})
}

// Delete removes a note by id.
func Delete(s *store.Store, id int) error {
	return s.Use(func(d *store.Data) error {
		for i := range d.Notes {
			if d.Notes[i].ID == id {
				d.Notes = append(d.Notes[:i], d.Notes[i+1:]...)
				return nil
			}
		}
		return fmt.Errorf("note %d not found", id)
	})
}

// Pin toggles the pinned flag on a note.
func Pin(s *store.Store, id int, pinned bool) error {
	return s.Use(func(d *store.Data) error {
		for i := range d.Notes {
			if d.Notes[i].ID == id {
				d.Notes[i].Pinned = pinned
				d.Notes[i].UpdatedAt = time.Now()
				return nil
			}
		}
		return fmt.Errorf("note %d not found", id)
	})
}

// Get returns a copy of one note.
func Get(s *store.Store, id int) (store.Note, bool) {
	var found store.Note
	var ok bool
	s.View(func(d *store.Data) {
		for _, n := range d.Notes {
			if n.ID == id {
				found = n
				ok = true
				return
			}
		}
	})
	return found, ok
}

// Filter returns notes matching all of the given conditions.  An empty
// query returns everything.  Tags are AND'd.  Pinned notes always
// appear first when sortByPinned is true.
type FilterOpts struct {
	Query        string
	Tags         []string
	IncludeBody  bool
	OnlyPinned   bool
	SortByPinned bool
	SortByDate   bool
}

func Filter(s *store.Store, opts FilterOpts) []store.Note {
	q := strings.ToLower(strings.TrimSpace(opts.Query))
	tags := normalizeTags(opts.Tags)

	var out []store.Note
	s.View(func(d *store.Data) {
		for _, n := range d.Notes {
			if opts.OnlyPinned && !n.Pinned {
				continue
			}
			if !hasAllTags(n.Tags, tags) {
				continue
			}
			if q != "" {
				hay := strings.ToLower(n.Title)
				if opts.IncludeBody {
					hay += "\n" + strings.ToLower(n.Body)
				}
				if !strings.Contains(hay, q) {
					continue
				}
			}
			out = append(out, n)
		}
	})

	sort.SliceStable(out, func(i, j int) bool {
		if opts.SortByPinned && out[i].Pinned != out[j].Pinned {
			return out[i].Pinned
		}
		if opts.SortByDate {
			return out[i].UpdatedAt.After(out[j].UpdatedAt)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// AllTags returns every tag present in the store with usage count.
func AllTags(s *store.Store) map[string]int {
	counts := map[string]int{}
	s.View(func(d *store.Data) {
		for _, n := range d.Notes {
			for _, t := range n.Tags {
				counts[t]++
			}
		}
	})
	return counts
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

func hasAllTags(have, need []string) bool {
	if len(need) == 0 {
		return true
	}
	set := map[string]struct{}{}
	for _, t := range have {
		set[t] = struct{}{}
	}
	for _, t := range need {
		if _, ok := set[t]; !ok {
			return false
		}
	}
	return true
}
