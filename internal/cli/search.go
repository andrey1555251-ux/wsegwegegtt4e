package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/andrey1555251-ux/mindforge/internal/store"
	"github.com/andrey1555251-ux/mindforge/internal/ui"
)

// runSearch performs a single substring search across every kind of
// record (notes, tasks, journal sections) and prints a unified hit
// list, much like a tiny `rg` for your data file.
func runSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	limit := fs.Int("limit", 50, "maximum hits per kind")
	caseSens := fs.Bool("case", false, "case-sensitive search")
	args = reorderArgs(args, []string{"limit"})
	if err := fs.Parse(args); err != nil {
		return err
	}
	q := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if q == "" {
		return fmt.Errorf("usage: mindforge search <query>")
	}
	probe := q
	if !*caseSens {
		probe = strings.ToLower(q)
	}
	st, err := open()
	if err != nil {
		return err
	}
	d, err := st.Snapshot()
	if err != nil {
		return err
	}

	type hit struct {
		kind, line string
	}
	var hits []hit

	addNote := func(n store.Note) {
		hits = append(hits, hit{"note", fmt.Sprintf("#%d  %s  %s", n.ID, n.Title, ui.Dim(n.UpdatedAt.Local().Format("Jan 02")))})
	}
	addTask := func(t store.Task) {
		mark := " "
		if t.Done {
			mark = ui.Green("v")
		}
		hits = append(hits, hit{"task", fmt.Sprintf("%s #%d  P%d  %s", mark, t.ID, t.Priority, t.Title)})
	}
	addJournal := func(date string, body string) {
		hits = append(hits, hit{"journal", fmt.Sprintf("%s  %s", ui.Dim(date), truncate(body, 80))})
	}

	matches := func(s string) bool {
		hay := s
		if !*caseSens {
			hay = strings.ToLower(s)
		}
		return strings.Contains(hay, probe)
	}

	count := map[string]int{}
	for _, n := range d.Notes {
		if matches(n.Title) || matches(n.Body) || matches(strings.Join(n.Tags, " ")) {
			if count["note"] < *limit {
				addNote(n)
			}
			count["note"]++
		}
	}
	for _, t := range d.Tasks {
		if matches(t.Title) || matches(t.Notes) || matches(strings.Join(t.Tags, " ")) {
			if count["task"] < *limit {
				addTask(t)
			}
			count["task"]++
		}
	}
	for _, e := range d.Journal {
		for _, sec := range e.Sections {
			if matches(sec.Body) {
				if count["journal"] < *limit {
					addJournal(e.Date, sec.Body)
				}
				count["journal"]++
			}
		}
	}

	if len(hits) == 0 {
		fmt.Println(ui.Dim("(no matches for ")+ui.Italic(q)+ui.Dim(")"))
		return nil
	}
	prev := ""
	for _, h := range hits {
		if h.kind != prev {
			fmt.Fprintln(os.Stdout)
			ui.PrintHeader(os.Stdout, strings.ToUpper(h.kind))
			prev = h.kind
		}
		fmt.Fprintln(os.Stdout, "  "+h.line)
	}

	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "%s  %s notes · %s tasks · %s journal sections\n",
		ui.Dim("totals:"),
		ui.Bold(itoa(count["note"])),
		ui.Bold(itoa(count["task"])),
		ui.Bold(itoa(count["journal"])),
	)
	return nil
}
