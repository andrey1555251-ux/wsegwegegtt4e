package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
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
