package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/andrey1555251-ux/mindforge/internal/notes"
	"github.com/andrey1555251-ux/mindforge/internal/store"
	"github.com/andrey1555251-ux/mindforge/internal/tasks"
	"github.com/andrey1555251-ux/mindforge/internal/ui"
)

// runQuickCapture handles the `mindforge add ...` shortcut.  It looks
// at the leading prefix to decide whether to create a task or a note:
//
//	mindforge add buy milk            -> task
//	mindforge add @ note: refactor    -> note (`@` prefix)
//	mindforge add ! buy milk          -> high-priority task (`!` prefix)
//	mindforge add "buy milk #home"    -> task with tag #home
func runQuickCapture(args []string) error {
	text := strings.TrimSpace(strings.Join(args, " "))
	if text == "" {
		return errors.New("usage: mindforge add <text>")
	}
	prefix := text[0]
	rest := strings.TrimSpace(text[1:])
	st, err := open()
	if err != nil {
		return err
	}
	switch prefix {
	case '@':
		title, body := splitFirstColon(rest)
		id, err := notes.Add(st, title, body, extractInlineTags(rest))
		if err != nil {
			return err
		}
		fmt.Println(ui.Green("note saved"), ui.Dim("#"+itoa(id)))
		return nil
	case '!':
		title, tagList := stripInlineTags(rest)
		id, err := tasks.Add(st, title, 1, nil, tagList, "")
		if err != nil {
			return err
		}
		fmt.Println(ui.Green("urgent task added"), ui.Dim("#"+itoa(id)))
		return nil
	}
	title, tagList := stripInlineTags(text)
	id, err := tasks.Add(st, title, 3, nil, tagList, "")
	if err != nil {
		return err
	}
	fmt.Println(ui.Green("task added"), ui.Dim("#"+itoa(id)))
	return nil
}

// stripInlineTags moves any "#word" tokens out of s into a slice and
// returns the cleaned-up sentence.
func stripInlineTags(s string) (string, []string) {
	fields := strings.Fields(s)
	var keep []string
	var tags []string
	for _, f := range fields {
		if strings.HasPrefix(f, "#") && len(f) > 1 {
			tags = append(tags, strings.TrimPrefix(f, "#"))
			continue
		}
		keep = append(keep, f)
	}
	return strings.Join(keep, " "), tags
}

func extractInlineTags(s string) []string {
	_, tags := stripInlineTags(s)
	return tags
}

func splitFirstColon(s string) (title, body string) {
	if i := strings.Index(s, ":"); i >= 0 {
		return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:])
	}
	return s, ""
}

// runCalendar prints a small ASCII month view.  It marks days that have
// journal entries with `j`, days with completed pomodoros with `*`,
// days that have due tasks with `!`, and overlays them in colour.
func runCalendar(args []string) error {
	now := time.Now()
	if len(args) > 0 {
		if t, err := time.Parse("2006-01", args[0]); err == nil {
			now = t
		}
	}
	st, err := open()
	if err != nil {
		return err
	}
	d, err := st.Snapshot()
	if err != nil {
		return err
	}

	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	next := first.AddDate(0, 1, 0)

	hasJournal := map[string]bool{}
	for _, e := range d.Journal {
		if len(e.Sections) > 0 {
			hasJournal[e.Date] = true
		}
	}
	pomCount := map[string]int{}
	for _, p := range d.Pomodoros {
		if p.Interrupted {
			continue
		}
		pomCount[p.StartedAt.Format("2006-01-02")]++
	}
	dueCount := map[string]int{}
	for _, t := range d.Tasks {
		if t.Due == nil || t.Done {
			continue
		}
		dueCount[t.Due.Format("2006-01-02")]++
	}

	ui.PrintHeader(os.Stdout, "Calendar — "+first.Format("January 2006"))
	fmt.Println(ui.Dim("Mo Tu We Th Fr Sa Su"))

	weekday := int(first.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	for i := 1; i < weekday; i++ {
		fmt.Print("   ")
	}
	for cur := first; cur.Before(next); cur = cur.AddDate(0, 0, 1) {
		key := cur.Format("2006-01-02")
		cell := fmt.Sprintf("%2d", cur.Day())
		marker := " "
		switch {
		case dueCount[key] > 0:
			marker = ui.Red("!")
		case pomCount[key] > 0:
			marker = ui.Magenta("*")
		case hasJournal[key]:
			marker = ui.Cyan("j")
		}
		if cur.Format("2006-01-02") == time.Now().Format("2006-01-02") {
			cell = ui.Bold(ui.Green(cell))
		}
		fmt.Print(cell + marker)
		w := int(cur.Weekday())
		if w == 0 {
			fmt.Println()
		} else {
			fmt.Print(" ")
		}
	}
	fmt.Println()
	fmt.Println(ui.Dim("legend: ! task due  · * pomodoros  · j journal entry"))
	return nil
}

// runAgenda prints tasks grouped by due date over the next N days,
// followed by a "later" bucket for due dates further out.
func runAgenda(args []string) error {
	days := 7
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--days", "-d":
			if i+1 >= len(args) {
				return errors.New("--days needs a value")
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil {
				return err
			}
			days = n
			i++
		}
	}
	st, err := open()
	if err != nil {
		return err
	}
	open := tasks.List(st, tasks.FilterOpts{})
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	cutoff := today.AddDate(0, 0, days)

	overdue := []store.Task{}
	bucket := map[string][]store.Task{}
	noDue := []store.Task{}
	later := []store.Task{}
	for _, t := range open {
		if t.Due == nil {
			noDue = append(noDue, t)
			continue
		}
		due := t.Due.In(now.Location())
		dueDay := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, now.Location())
		switch {
		case dueDay.Before(today):
			overdue = append(overdue, t)
		case !dueDay.Before(cutoff):
			later = append(later, t)
		default:
			key := dueDay.Format("2006-01-02")
			bucket[key] = append(bucket[key], t)
		}
	}

	ui.PrintHeader(os.Stdout, fmt.Sprintf("Agenda — next %d days", days))
	if len(overdue) > 0 {
		fmt.Println(ui.Red("OVERDUE"))
		for _, t := range overdue {
			printAgendaTask(t, now)
		}
		fmt.Println()
	}
	any := false
	for i := 0; i < days; i++ {
		day := today.AddDate(0, 0, i)
		key := day.Format("2006-01-02")
		ts := bucket[key]
		if len(ts) == 0 {
			continue
		}
		any = true
		header := day.Format("Mon · 2006-01-02")
		if i == 0 {
			header += "  " + ui.Dim("(today)")
		} else if i == 1 {
			header += "  " + ui.Dim("(tomorrow)")
		}
		fmt.Println(ui.Bold(header))
		for _, t := range ts {
			printAgendaTask(t, now)
		}
		fmt.Println()
	}
	if !any && len(overdue) == 0 {
		fmt.Println(ui.Dim("nothing scheduled in this window."))
	}
	if len(later) > 0 {
		fmt.Println(ui.Bold("Later"))
		for _, t := range later {
			printAgendaTask(t, now)
		}
		fmt.Println()
	}
	if len(noDue) > 0 {
		fmt.Println(ui.Dim(fmt.Sprintf("(%d open tasks have no due date)", len(noDue))))
	}
	return nil
}

func printAgendaTask(t store.Task, now time.Time) {
	pri := ui.Cyan(fmt.Sprintf("P%d", t.Priority))
	tail := ""
	if t.Due != nil {
		tail = " " + ui.Dim(formatDueShort(*t.Due, now))
	}
	if len(t.Tags) > 0 {
		tail += " " + renderTags(t.Tags)
	}
	fmt.Printf("  %s  #%d  %s%s\n", pri, t.ID, t.Title, tail)
}

// runSummary prints a markdown-friendly recap covering the last N
// days.  Designed to be pasteable into a status update or weekly
// review.  It deliberately uses no ANSI colours so it survives a copy.
func runSummary(args []string) error {
	days := 7
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--days", "-d":
			if i+1 >= len(args) {
				return errors.New("--days needs a value")
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil || n <= 0 {
				return fmt.Errorf("invalid --days %q", args[i+1])
			}
			days = n
			i++
		case "--week":
			days = 7
		case "--month":
			days = 30
		}
	}
	st, err := open()
	if err != nil {
		return err
	}
	d, err := st.Snapshot()
	if err != nil {
		return err
	}
	now := time.Now()
	since := now.AddDate(0, 0, -days)

	w := os.Stdout
	fmt.Fprintf(w, "# MindForge summary — last %d days\n", days)
	fmt.Fprintf(w, "_%s → %s_\n\n", since.Format("Mon 02 Jan"), now.Format("Mon 02 Jan 2006"))

	// Tasks completed in the window.
	var completed []store.Task
	for _, t := range d.Tasks {
		if t.Done && t.DoneAt != nil && t.DoneAt.After(since) {
			completed = append(completed, t)
		}
	}
	fmt.Fprintf(w, "## ✓ Completed (%d)\n", len(completed))
	if len(completed) == 0 {
		fmt.Fprintln(w, "_no tasks closed in this window_")
	} else {
		for _, t := range completed {
			fmt.Fprintf(w, "- [x] **P%d** %s", t.Priority, t.Title)
			if len(t.Tags) > 0 {
				fmt.Fprintf(w, "  _#%s_", strings.Join(t.Tags, " #"))
			}
			fmt.Fprintln(w)
		}
	}

	// Open and overdue.
	var overdue []store.Task
	openCount := 0
	for _, t := range d.Tasks {
		if t.Done || t.Archived {
			continue
		}
		openCount++
		if t.Due != nil && t.Due.Before(now) {
			overdue = append(overdue, t)
		}
	}
	fmt.Fprintf(w, "\n## ⏳ Open (%d) · overdue (%d)\n", openCount, len(overdue))
	for _, t := range overdue {
		fmt.Fprintf(w, "- [ ] **P%d** %s — _due %s_\n", t.Priority, t.Title, t.Due.Format("2006-01-02"))
	}

	// Pomodoros + focus.
	pomCount := 0
	focusMin := 0
	byLabel := map[string]int{}
	for _, p := range d.Pomodoros {
		if p.Interrupted || !p.StartedAt.After(since) {
			continue
		}
		pomCount++
		focusMin += p.Minutes
		k := p.Label
		if k == "" {
			k = "(unlabeled)"
		}
		byLabel[k] += p.Minutes
	}
	fmt.Fprintf(w, "\n## 🍅 Focus\n")
	fmt.Fprintf(w, "- %d pomodoros · %s\n", pomCount, formatMinutes(focusMin))
	if len(byLabel) > 0 {
		type kv struct {
			k string
			v int
		}
		var rows []kv
		for k, v := range byLabel {
			rows = append(rows, kv{k, v})
		}
		// simple insertion sort by minutes desc
		for i := 1; i < len(rows); i++ {
			for j := i; j > 0 && rows[j].v > rows[j-1].v; j-- {
				rows[j], rows[j-1] = rows[j-1], rows[j]
			}
		}
		for _, r := range rows {
			fmt.Fprintf(w, "  - **%s** — %s\n", r.k, formatMinutes(r.v))
		}
	}

	// Journal entries + average mood.
	jc := 0
	moodSum, moodN := 0, 0
	cutoff := since.Format("2006-01-02")
	for _, e := range d.Journal {
		if e.Date < cutoff {
			continue
		}
		jc++
		if e.Mood >= 1 && e.Mood <= 5 {
			moodSum += e.Mood
			moodN++
		}
	}
	fmt.Fprintf(w, "\n## 📓 Journal\n")
	fmt.Fprintf(w, "- %d days written\n", jc)
	if moodN > 0 {
		fmt.Fprintf(w, "- avg mood: %.2f / 5 _(across %d entries)_\n", float64(moodSum)/float64(moodN), moodN)
	}

	// Habits — checks in window.
	if len(d.Habits) > 0 {
		fmt.Fprintf(w, "\n## 🌱 Habits\n")
		for _, h := range d.Habits {
			c := 0
			for _, day := range h.Checks {
				if day >= cutoff {
					c++
				}
			}
			fmt.Fprintf(w, "- **%s** — %d / %d days\n", h.Name, c, days)
		}
	}

	// New notes in window.
	nn := 0
	for _, n := range d.Notes {
		if n.CreatedAt.After(since) {
			nn++
		}
	}
	fmt.Fprintf(w, "\n## 🗒  Notes\n- %d new note(s) in this window\n", nn)
	return nil
}

// runEditDataFile opens the JSON data file in the user's preferred
// editor.  This is a power-user escape hatch for fixing weird state.
func runEditDataFile() error {
	st, err := open()
	if err != nil {
		return err
	}
	editor := pickEditor()
	if editor == "" {
		return fmt.Errorf("no editor found — set EDITOR or VISUAL")
	}
	cmd := exec.Command(editor, st.Path())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor exited: %w", err)
	}
	// Reload to validate.
	if _, err := store.Open(st.Path()); err != nil {
		return fmt.Errorf("reload after edit: %w", err)
	}
	fmt.Println(ui.Green("ok — edits saved and reloaded"))
	return nil
}

func pickEditor() string {
	for _, key := range []string{"VISUAL", "EDITOR"} {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	if runtime.GOOS == "windows" {
		return "notepad"
	}
	for _, candidate := range []string{"nano", "vim", "vi"} {
		if _, err := exec.LookPath(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

// suppress "imported and not used" for io if we ever drop the var.
var _ io.Writer = os.Stdout
