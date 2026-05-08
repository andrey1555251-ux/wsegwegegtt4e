package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/andrey1555251-ux/mindforge/internal/store"
)

// writeJSON dumps the data exactly as it lives on disk — useful for
// piping into jq or another tool.
func writeJSON(w io.Writer, d *store.Data) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(d)
}

// writeMarkdown produces a human-readable digest grouped by section.
// It is suitable for committing into a personal repo or sharing with
// a friend.
func writeMarkdown(w io.Writer, d *store.Data) error {
	fmt.Fprintln(w, "# MindForge export")
	fmt.Fprintln(w, "_generated", time.Now().Format(time.RFC3339)+"_")
	fmt.Fprintln(w)

	if len(d.Notes) > 0 {
		fmt.Fprintln(w, "## Notes")
		notes := append([]store.Note(nil), d.Notes...)
		sort.SliceStable(notes, func(i, j int) bool {
			if notes[i].Pinned != notes[j].Pinned {
				return notes[i].Pinned
			}
			return notes[i].UpdatedAt.After(notes[j].UpdatedAt)
		})
		for _, n := range notes {
			pin := ""
			if n.Pinned {
				pin = " *(pinned)*"
			}
			fmt.Fprintf(w, "### %s%s\n", n.Title, pin)
			if len(n.Tags) > 0 {
				fmt.Fprintf(w, "_tags:_ %s\n\n", strings.Join(prefixTags(n.Tags), " "))
			}
			if n.Body != "" {
				fmt.Fprintln(w, n.Body)
			}
			fmt.Fprintf(w, "\n_id %d · created %s · updated %s_\n\n",
				n.ID,
				n.CreatedAt.Format(time.RFC3339),
				n.UpdatedAt.Format(time.RFC3339))
		}
	}

	if len(d.Tasks) > 0 {
		fmt.Fprintln(w, "## Tasks")
		for _, t := range d.Tasks {
			mark := "[ ]"
			if t.Done {
				mark = "[x]"
			}
			due := ""
			if t.Due != nil {
				due = " _(due " + t.Due.Format("2006-01-02") + ")_"
			}
			fmt.Fprintf(w, "- %s **P%d** %s%s\n", mark, t.Priority, t.Title, due)
			if t.Notes != "" {
				for _, line := range strings.Split(t.Notes, "\n") {
					fmt.Fprintf(w, "    %s\n", line)
				}
			}
		}
		fmt.Fprintln(w)
	}

	if len(d.Journal) > 0 {
		fmt.Fprintln(w, "## Journal")
		entries := append([]store.JournalEntry(nil), d.Journal...)
		sort.SliceStable(entries, func(i, j int) bool { return entries[i].Date > entries[j].Date })
		for _, e := range entries {
			fmt.Fprintf(w, "### %s\n", e.Date)
			if e.Mood > 0 {
				fmt.Fprintf(w, "_mood %d/5_\n\n", e.Mood)
			}
			for _, sec := range e.Sections {
				fmt.Fprintf(w, "**%s**\n\n%s\n\n", sec.At.Format("15:04"), sec.Body)
			}
		}
	}

	return nil
}

// writeCSV writes a simple multi-row dump where each row prefixes its
// kind so a single CSV file is enough for a spreadsheet user.
func writeCSV(w io.Writer, d *store.Data) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()
	if err := cw.Write([]string{"kind", "id", "title", "tags", "body", "extra", "created_at"}); err != nil {
		return err
	}
	for _, n := range d.Notes {
		if err := cw.Write([]string{
			"note",
			fmt.Sprintf("%d", n.ID),
			n.Title,
			strings.Join(n.Tags, ","),
			n.Body,
			fmt.Sprintf("pinned=%t", n.Pinned),
			n.CreatedAt.Format(time.RFC3339),
		}); err != nil {
			return err
		}
	}
	for _, t := range d.Tasks {
		due := ""
		if t.Due != nil {
			due = t.Due.Format(time.RFC3339)
		}
		if err := cw.Write([]string{
			"task",
			fmt.Sprintf("%d", t.ID),
			t.Title,
			strings.Join(t.Tags, ","),
			t.Notes,
			fmt.Sprintf("priority=%d done=%t due=%s", t.Priority, t.Done, due),
			t.CreatedAt.Format(time.RFC3339),
		}); err != nil {
			return err
		}
	}
	for _, e := range d.Journal {
		body := strings.Builder{}
		for _, sec := range e.Sections {
			body.WriteString(sec.At.Format("15:04 "))
			body.WriteString(sec.Body)
			body.WriteString("\n")
		}
		if err := cw.Write([]string{
			"journal",
			"",
			e.Date,
			"",
			body.String(),
			fmt.Sprintf("mood=%d", e.Mood),
			"",
		}); err != nil {
			return err
		}
	}
	for _, p := range d.Pomodoros {
		if err := cw.Write([]string{
			"pomodoro",
			"",
			p.Label,
			"",
			"",
			fmt.Sprintf("minutes=%d interrupted=%t finished=%s",
				p.Minutes, p.Interrupted, p.FinishedAt.Format(time.RFC3339)),
			p.StartedAt.Format(time.RFC3339),
		}); err != nil {
			return err
		}
	}
	return nil
}

func prefixTags(tags []string) []string {
	out := make([]string, len(tags))
	for i, t := range tags {
		out[i] = "#" + t
	}
	return out
}
