package cli

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/andrey1555251-ux/mindforge/internal/notes"
	"github.com/andrey1555251-ux/mindforge/internal/store"
	"github.com/andrey1555251-ux/mindforge/internal/tasks"
	"github.com/andrey1555251-ux/mindforge/internal/ui"
)

// runImport reads a file (or stdin) and creates notes/tasks from it.
// Three formats are supported:
//
//	--format text   one note per non-empty paragraph (default)
//	--format csv    a header row decides which fields land where
//	--format json   the same shape as `mindforge export --format json`
func runImport(args []string) error {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	format := fs.String("format", "text", "text | csv | json")
	kind := fs.String("kind", "note", "note | task (text/csv only)")
	tag := fs.String("tag", "", "tags to apply to every imported item")
	in := fs.String("in", "", "input file (default stdin)")
	args = reorderArgs(args, []string{"format", "kind", "tag", "in"})
	if err := fs.Parse(args); err != nil {
		return err
	}

	r, closer, err := openImportReader(*in)
	if err != nil {
		return err
	}
	if closer != nil {
		defer closer.Close()
	}

	st, err := open()
	if err != nil {
		return err
	}

	switch *format {
	case "text":
		return importText(st, r, *kind, splitTags(*tag))
	case "csv":
		return importCSV(st, r, *kind, splitTags(*tag))
	case "json":
		return importJSON(st, r)
	}
	return fmt.Errorf("unknown format %q", *format)
}

func openImportReader(path string) (io.Reader, io.Closer, error) {
	if path == "" {
		fi, err := os.Stdin.Stat()
		if err != nil {
			return nil, nil, err
		}
		if (fi.Mode() & os.ModeCharDevice) != 0 {
			return nil, nil, errors.New("nothing on stdin and --in not given")
		}
		return os.Stdin, nil, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	return f, f, nil
}

func importText(st *store.Store, r io.Reader, kind string, tags []string) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var (
		buf strings.Builder
		n   int
	)
	flush := func() error {
		body := strings.TrimSpace(buf.String())
		buf.Reset()
		if body == "" {
			return nil
		}
		title, rest := splitTitleBody(body)
		switch kind {
		case "note":
			if _, err := notes.Add(st, title, rest, tags); err != nil {
				return err
			}
		case "task":
			if _, err := tasks.Add(st, title, 3, nil, tags, rest); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown kind %q", kind)
		}
		n++
		return nil
	}
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			if err := flush(); err != nil {
				return err
			}
			continue
		}
		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(line)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if err := flush(); err != nil {
		return err
	}
	fmt.Println(ui.Green("imported"), n, kind+"(s)")
	return nil
}

func importCSV(st *store.Store, r io.Reader, kind string, tags []string) error {
	cr := csv.NewReader(r)
	cr.LazyQuotes = true
	header, err := cr.Read()
	if err != nil {
		return fmt.Errorf("reading csv header: %w", err)
	}
	col := map[string]int{}
	for i, h := range header {
		col[strings.ToLower(strings.TrimSpace(h))] = i
	}
	required := func(k string) (int, bool) {
		idx, ok := col[k]
		return idx, ok
	}
	titleIdx, ok := required("title")
	if !ok {
		return errors.New("csv must have a 'title' column")
	}
	bodyIdx := col["body"]
	tagsIdx, hasTags := col["tags"]
	priorityIdx, hasPriority := col["priority"]

	var n int
	for {
		row, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		title := safeAt(row, titleIdx)
		if title == "" {
			continue
		}
		body := safeAt(row, bodyIdx)
		merged := tags
		if hasTags {
			rowTags := splitTags(strings.ReplaceAll(safeAt(row, tagsIdx), " ", ","))
			merged = mergeTags(merged, rowTags)
		}
		switch kind {
		case "note":
			if _, err := notes.Add(st, title, body, merged); err != nil {
				return err
			}
		case "task":
			pri := 3
			if hasPriority {
				if v, err := strconv.Atoi(strings.TrimSpace(safeAt(row, priorityIdx))); err == nil {
					pri = v
				}
			}
			if _, err := tasks.Add(st, title, pri, nil, merged, body); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown kind %q", kind)
		}
		n++
	}
	fmt.Println(ui.Green("imported"), n, kind+"(s) from csv")
	return nil
}

func importJSON(st *store.Store, r io.Reader) error {
	raw, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	d := &store.Data{}
	if err := json.Unmarshal(raw, d); err != nil {
		return fmt.Errorf("parsing json: %w", err)
	}
	var nNotes, nTasks, nHabits int
	for _, note := range d.Notes {
		if _, err := notes.Add(st, note.Title, note.Body, note.Tags); err != nil {
			return err
		}
		nNotes++
	}
	for _, t := range d.Tasks {
		due := t.Due
		if _, err := tasks.Add(st, t.Title, defInt(t.Priority, 3), due, t.Tags, t.Notes); err != nil {
			return err
		}
		nTasks++
	}
	if len(d.Habits) > 0 {
		err := st.Use(func(data *store.Data) error {
			for _, h := range d.Habits {
				if data.Counters.NextHabitID == 0 {
					data.Counters.NextHabitID = 1
				}
				h.ID = data.Counters.NextHabitID
				data.Counters.NextHabitID++
				data.Habits = append(data.Habits, h)
				nHabits++
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	fmt.Printf("%s %d notes · %d tasks · %d habits\n",
		ui.Green("imported"), nNotes, nTasks, nHabits)
	return nil
}

func splitTitleBody(s string) (string, string) {
	s = strings.TrimSpace(s)
	idx := strings.Index(s, "\n")
	if idx == -1 {
		return s, ""
	}
	return strings.TrimSpace(s[:idx]), strings.TrimSpace(s[idx+1:])
}

func safeAt(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return row[idx]
}

func mergeTags(a, b []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range append(a, b...) {
		t = strings.TrimSpace(t)
		if t == "" || seen[strings.ToLower(t)] {
			continue
		}
		seen[strings.ToLower(t)] = true
		out = append(out, t)
	}
	return out
}

func defInt(v, fallback int) int {
	if v == 0 {
		return fallback
	}
	return v
}
