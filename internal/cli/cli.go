// Package cli wires the user-facing commands.  It tries hard to feel
// like a single, cohesive tool: every command accepts -h/--help, every
// destructive action prints what it changed, and the help text is the
// same string everywhere so there is one source of truth.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/andrey1555251-ux/mindforge/internal/journal"
	"github.com/andrey1555251-ux/mindforge/internal/notes"
	"github.com/andrey1555251-ux/mindforge/internal/pomodoro"
	"github.com/andrey1555251-ux/mindforge/internal/stats"
	"github.com/andrey1555251-ux/mindforge/internal/store"
	"github.com/andrey1555251-ux/mindforge/internal/tasks"
	"github.com/andrey1555251-ux/mindforge/internal/ui"
	"github.com/andrey1555251-ux/mindforge/internal/version"
)

// Run is the entry point used by main().  It dispatches argv[0] to the
// right sub-command.  We deliberately keep the dispatcher in one place
// so the help text is easy to scan.
func Run(args []string) error {
	if len(args) == 0 {
		return runWelcome(os.Stdout)
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "-h", "--help", "help":
		return runHelp(os.Stdout)
	case "-v", "--version", "version":
		fmt.Println(version.Version)
		return nil
	case "welcome":
		return runWelcome(os.Stdout)
	case "note", "notes", "n":
		return runNote(rest)
	case "task", "tasks", "t":
		return runTask(rest)
	case "journal", "j":
		return runJournal(rest)
	case "pomodoro", "pomo", "p":
		return runPomodoro(rest)
	case "stats", "status", "s":
		return runStats(rest)
	case "tags":
		return runTags(rest)
	case "where", "path":
		return runWhere(rest)
	case "backup":
		return runBackup(rest)
	case "export":
		return runExport(rest)
	case "config":
		return runConfig(rest)
	case "quote":
		return runQuote(os.Stdout)
	case "doctor":
		return runDoctor(os.Stdout)
	case "today":
		return runToday(os.Stdout)
	case "search", "find":
		return runSearch(rest)
	case "add":
		return runQuickCapture(rest)
	case "calendar", "cal":
		return runCalendar(rest)
	case "edit-data":
		return runEditDataFile()
	case "habit", "habits", "h":
		return runHabit(rest)
	case "import":
		return runImport(rest)
	case "vault":
		return runVault(rest)
	case "motd":
		return runMotd(os.Stdout)
	}
	return fmt.Errorf("unknown command %q — try `mindforge help`", cmd)
}

// open is a tiny convenience helper that obtains the default store and
// translates the common "permission denied" error into a more useful
// hint.
func open() (*store.Store, error) {
	st, err := store.Default()
	if err != nil {
		return nil, fmt.Errorf("opening data file: %w", err)
	}
	return st, nil
}

func runWelcome(w io.Writer) error {
	fmt.Fprint(w, ui.Banner(version.Version))
	fmt.Fprintln(w, ui.Bold("Welcome!"))
	fmt.Fprintln(w, "MindForge keeps your notes, tasks and focus time in a single place.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, ui.Bold("Try one of:"))
	fmt.Fprintln(w, "  ", ui.Cyan("mindforge note add"), "  -t \"my first note\"")
	fmt.Fprintln(w, "  ", ui.Cyan("mindforge task add"), "  \"finish the report\" --due tomorrow --priority 1")
	fmt.Fprintln(w, "  ", ui.Cyan("mindforge pomodoro"), " --label \"deep work\" --rounds 4")
	fmt.Fprintln(w, "  ", ui.Cyan("mindforge today"), "      summary of what's on the agenda")
	fmt.Fprintln(w, "  ", ui.Cyan("mindforge stats"), "      streaks and totals")
	fmt.Fprintln(w)
	fmt.Fprintln(w, ui.Dim("data lives at "+mustPath()))
	return nil
}

func mustPath() string {
	st, err := store.Default()
	if err != nil {
		return "?"
	}
	return st.Path()
}

func runHelp(w io.Writer) error {
	fmt.Fprint(w, ui.Banner(version.Version))
	fmt.Fprintln(w, ui.Bold("Usage:"), "mindforge <command> [...]")
	fmt.Fprintln(w)
	groups := []struct {
		title string
		rows  [][2]string
	}{
		{"Notes", [][2]string{
			{"note add", "create a new note (-t title -b body --tag foo)"},
			{"note list", "list notes (--tag foo --query word --pinned)"},
			{"note show <id>", "show one note in full"},
			{"note edit <id>", "edit title/body/tags"},
			{"note pin <id>", "pin or unpin a note (--off to unpin)"},
			{"note delete <id>", "delete a note"},
		}},
		{"Tasks", [][2]string{
			{"task add", "add a task (\"title\" --priority 1 --due 2026-12-31 --tag work)"},
			{"task list", "list open tasks (--all --done --overdue --soon 24)"},
			{"task done <id>", "mark task done"},
			{"task reopen <id>", "reopen a previously completed task"},
			{"task edit <id>", "edit a task"},
			{"task delete <id>", "delete a task"},
		}},
		{"Journal", [][2]string{
			{"journal write", "append text to today's entry (-b ...)"},
			{"journal mood", "record today's mood 1..5 (--mood 4)"},
			{"journal show", "show today's entry"},
			{"journal log", "show recent entries (--days 14)"},
		}},
		{"Pomodoro", [][2]string{
			{"pomodoro", "start a session (--label .. --rounds 4 --focus 25)"},
		}},
		{"Insight", [][2]string{
			{"today", "a curated daily briefing"},
			{"stats", "totals, streaks and a tiny heatmap"},
			{"tags", "show every tag with usage counts"},
			{"quote", "a random encouragement"},
		}},
		{"Maintenance", [][2]string{
			{"where", "print where your data lives"},
			{"backup", "make a timestamped backup"},
			{"export", "dump everything (--format json|md)"},
			{"config", "view or change settings (--get|--set k=v)"},
			{"doctor", "self-check the data file"},
			{"version", "show version"},
		}},
	}
	for _, g := range groups {
		fmt.Fprintln(w, ui.Bold(ui.Cyan(g.title)))
		for _, r := range g.rows {
			fmt.Fprintf(w, "  %s  %s\n", ui.Pad(r[0], 18), ui.Dim(r[1]))
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, ui.Dim("data lives at "+mustPath()))
	return nil
}

// ----------------------------------------------------------------------------
// notes
// ----------------------------------------------------------------------------

func runNote(args []string) error {
	if len(args) == 0 {
		return runNoteList(nil)
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "add", "new":
		return runNoteAdd(rest)
	case "list", "ls":
		return runNoteList(rest)
	case "show", "view":
		return runNoteShow(rest)
	case "edit":
		return runNoteEdit(rest)
	case "pin":
		return runNotePin(rest, true)
	case "unpin":
		return runNotePin(rest, false)
	case "delete", "rm", "del":
		return runNoteDelete(rest)
	}
	return fmt.Errorf("unknown note subcommand %q", cmd)
}

func runNoteAdd(args []string) error {
	fs := flag.NewFlagSet("note add", flag.ContinueOnError)
	title := fs.String("t", "", "title (required)")
	body := fs.String("b", "", "body text (or pipe via stdin)")
	tagStr := fs.String("tag", "", "comma-separated tags")
	args = reorderArgs(args, []string{"t", "b", "tag"})
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *title == "" && len(fs.Args()) > 0 {
		*title = strings.Join(fs.Args(), " ")
	}
	if *title == "" {
		return errors.New("title (-t) is required")
	}
	if *body == "" {
		if data, err := readStdinIfPiped(); err == nil && data != "" {
			*body = data
		}
	}
	st, err := open()
	if err != nil {
		return err
	}
	id, err := notes.Add(st, *title, *body, splitTags(*tagStr))
	if err != nil {
		return err
	}
	fmt.Println(ui.Green("note saved"), ui.Dim("#"+itoa(id)))
	return nil
}

func runNoteList(args []string) error {
	fs := flag.NewFlagSet("note list", flag.ContinueOnError)
	tag := fs.String("tag", "", "filter by comma-separated tags")
	q := fs.String("query", "", "case-insensitive substring search")
	body := fs.Bool("body", false, "search in body too")
	pinned := fs.Bool("pinned", false, "only pinned")
	all := fs.Bool("all", false, "no sort by pinned")
	if err := fs.Parse(args); err != nil {
		return err
	}
	st, err := open()
	if err != nil {
		return err
	}
	out := notes.Filter(st, notes.FilterOpts{
		Query:        *q,
		Tags:         splitTags(*tag),
		IncludeBody:  *body,
		OnlyPinned:   *pinned,
		SortByPinned: !*all,
		SortByDate:   true,
	})
	if len(out) == 0 {
		fmt.Println(ui.Dim("(no notes)"))
		return nil
	}
	rows := make([][]string, 0, len(out))
	for _, n := range out {
		marker := " "
		if n.Pinned {
			marker = ui.Yellow("*")
		}
		rows = append(rows, []string{
			marker,
			"#" + itoa(n.ID),
			truncate(n.Title, 50),
			renderTags(n.Tags),
			n.UpdatedAt.Local().Format("Jan 02 15:04"),
		})
	}
	ui.PrintTable(os.Stdout, []string{"", "id", "title", "tags", "updated"}, rows)
	return nil
}

func runNoteShow(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: mindforge note show <id>")
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid id %q", args[0])
	}
	st, err := open()
	if err != nil {
		return err
	}
	n, ok := notes.Get(st, id)
	if !ok {
		return fmt.Errorf("note %d not found", id)
	}
	ui.PrintHeader(os.Stdout, n.Title)
	if n.Pinned {
		fmt.Println(ui.Yellow("(pinned)"))
	}
	if len(n.Tags) > 0 {
		fmt.Println(renderTags(n.Tags))
	}
	if n.Body != "" {
		fmt.Println()
		fmt.Println(n.Body)
	}
	fmt.Println()
	fmt.Println(ui.Dim(fmt.Sprintf("created %s · updated %s",
		n.CreatedAt.Local().Format(time.RFC822),
		n.UpdatedAt.Local().Format(time.RFC822))))
	return nil
}

func runNoteEdit(args []string) error {
	fs := flag.NewFlagSet("note edit", flag.ContinueOnError)
	title := fs.String("t", "", "new title")
	body := fs.String("b", "", "new body")
	addTag := fs.String("tag", "", "tags to add (comma-separated)")
	setTag := fs.String("set-tag", "", "replace tags entirely")
	args = reorderArgs(args, []string{"t", "b", "tag", "set-tag"})
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return errors.New("usage: mindforge note edit <id> [-t title] [-b body] [--tag a,b]")
	}
	id, err := strconv.Atoi(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("invalid id %q", fs.Arg(0))
	}
	st, err := open()
	if err != nil {
		return err
	}
	var (
		tp *string
		bp *string
	)
	if fs.Lookup("t").Value.String() != "" {
		tp = title
	}
	if fs.Lookup("b").Value.String() != "" {
		bp = body
	}
	tags := splitTags(*addTag)
	replace := false
	if *setTag != "" {
		tags = splitTags(*setTag)
		replace = true
	}
	if err := notes.Update(st, id, tp, bp, tags, replace); err != nil {
		return err
	}
	fmt.Println(ui.Green("note updated"))
	return nil
}

func runNotePin(args []string, on bool) error {
	if len(args) == 0 {
		return errors.New("usage: mindforge note pin <id> [--off]")
	}
	pin := on
	for _, a := range args[1:] {
		if a == "--off" {
			pin = false
		}
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid id %q", args[0])
	}
	st, err := open()
	if err != nil {
		return err
	}
	if err := notes.Pin(st, id, pin); err != nil {
		return err
	}
	if pin {
		fmt.Println(ui.Green("pinned"))
	} else {
		fmt.Println(ui.Green("unpinned"))
	}
	return nil
}

func runNoteDelete(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: mindforge note delete <id>")
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid id %q", args[0])
	}
	st, err := open()
	if err != nil {
		return err
	}
	if err := notes.Delete(st, id); err != nil {
		return err
	}
	fmt.Println(ui.Green("note deleted"))
	return nil
}

// ----------------------------------------------------------------------------
// tasks
// ----------------------------------------------------------------------------

func runTask(args []string) error {
	if len(args) == 0 {
		return runTaskList(nil)
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "add", "new":
		return runTaskAdd(rest)
	case "list", "ls":
		return runTaskList(rest)
	case "done", "do":
		return runTaskMark(rest, true)
	case "reopen":
		return runTaskMark(rest, false)
	case "edit":
		return runTaskEdit(rest)
	case "delete", "rm", "del":
		return runTaskDelete(rest)
	}
	return fmt.Errorf("unknown task subcommand %q", cmd)
}

func runTaskAdd(args []string) error {
	fs := flag.NewFlagSet("task add", flag.ContinueOnError)
	priority := fs.Int("priority", 3, "priority 1..5 (1 highest)")
	dueRaw := fs.String("due", "", "due date (YYYY-MM-DD, 'today', 'tomorrow', '+3d')")
	tag := fs.String("tag", "", "comma-separated tags")
	body := fs.String("notes", "", "additional notes")
	args = reorderArgs(args, []string{"priority", "due", "tag", "notes"})
	if err := fs.Parse(args); err != nil {
		return err
	}
	title := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if title == "" {
		return errors.New("title is required")
	}
	due, err := parseDateMaybe(*dueRaw)
	if err != nil {
		return err
	}
	st, err := open()
	if err != nil {
		return err
	}
	id, err := tasks.Add(st, title, *priority, due, splitTags(*tag), *body)
	if err != nil {
		return err
	}
	fmt.Println(ui.Green("task added"), ui.Dim("#"+itoa(id)))
	return nil
}

func runTaskList(args []string) error {
	fs := flag.NewFlagSet("task list", flag.ContinueOnError)
	all := fs.Bool("all", false, "include completed")
	done := fs.Bool("done", false, "only completed")
	tag := fs.String("tag", "", "filter by tag")
	q := fs.String("query", "", "substring search")
	overdue := fs.Bool("overdue", false, "overdue only")
	soon := fs.Int("soon", 0, "due within N hours")
	if err := fs.Parse(args); err != nil {
		return err
	}
	st, err := open()
	if err != nil {
		return err
	}
	out := tasks.List(st, tasks.FilterOpts{
		IncludeDone: *all,
		OnlyDone:    *done,
		Tag:         *tag,
		Query:       *q,
		OverdueOnly: *overdue,
		DueSoonHrs:  *soon,
	})
	if len(out) == 0 {
		fmt.Println(ui.Dim("(no tasks)"))
		return nil
	}
	now := time.Now()
	rows := make([][]string, 0, len(out))
	for _, t := range out {
		status := " "
		if t.Done {
			status = ui.Green("v")
		} else if t.Due != nil && t.Due.Before(now) {
			status = ui.Red("!")
		}
		due := ""
		if t.Due != nil {
			due = formatDueShort(*t.Due, now)
		}
		title := t.Title
		if t.Done {
			title = ui.Dim(strikethrough(title))
		}
		rows = append(rows, []string{
			status,
			"#" + itoa(t.ID),
			"P" + itoa(t.Priority),
			truncate(title, 60),
			renderTags(t.Tags),
			due,
		})
	}
	ui.PrintTable(os.Stdout, []string{"", "id", "pri", "title", "tags", "due"}, rows)
	return nil
}

func runTaskMark(args []string, done bool) error {
	if len(args) == 0 {
		if done {
			return errors.New("usage: mindforge task done <id> [<id> ...]")
		}
		return errors.New("usage: mindforge task reopen <id> [<id> ...]")
	}
	st, err := open()
	if err != nil {
		return err
	}
	for _, raw := range args {
		id, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("invalid id %q", raw)
		}
		if done {
			if err := tasks.Done(st, id); err != nil {
				return err
			}
			fmt.Println(ui.Green("done"), ui.Dim("#"+itoa(id)))
		} else {
			if err := tasks.Reopen(st, id); err != nil {
				return err
			}
			fmt.Println(ui.Green("reopened"), ui.Dim("#"+itoa(id)))
		}
	}
	return nil
}

func runTaskEdit(args []string) error {
	fs := flag.NewFlagSet("task edit", flag.ContinueOnError)
	title := fs.String("t", "", "new title")
	body := fs.String("notes", "", "new notes")
	priority := fs.Int("priority", 0, "new priority 1..5 (0 = unchanged)")
	due := fs.String("due", "", "new due date or 'clear'")
	addTag := fs.String("tag", "", "tags to add")
	setTag := fs.String("set-tag", "", "replace tags entirely")
	args = reorderArgs(args, []string{"t", "notes", "priority", "due", "tag", "set-tag"})
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return errors.New("usage: mindforge task edit <id> ...")
	}
	id, err := strconv.Atoi(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("invalid id %q", fs.Arg(0))
	}
	var tp, bp *string
	if fs.Lookup("t").Value.String() != "" {
		tp = title
	}
	if fs.Lookup("notes").Value.String() != "" {
		bp = body
	}
	var pp *int
	if *priority != 0 {
		pp = priority
	}
	var dueT *time.Time
	clear := false
	if *due == "clear" {
		clear = true
	} else if *due != "" {
		t, err := parseDateMaybe(*due)
		if err != nil {
			return err
		}
		dueT = t
	}
	tags := splitTags(*addTag)
	replace := false
	if *setTag != "" {
		tags = splitTags(*setTag)
		replace = true
	}
	st, err := open()
	if err != nil {
		return err
	}
	if err := tasks.Update(st, id, tp, bp, pp, dueT, clear, tags, replace); err != nil {
		return err
	}
	fmt.Println(ui.Green("task updated"))
	return nil
}

func runTaskDelete(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: mindforge task delete <id>")
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid id %q", args[0])
	}
	st, err := open()
	if err != nil {
		return err
	}
	if err := tasks.Delete(st, id); err != nil {
		return err
	}
	fmt.Println(ui.Green("task deleted"))
	return nil
}

// ----------------------------------------------------------------------------
// journal
// ----------------------------------------------------------------------------

func runJournal(args []string) error {
	if len(args) == 0 {
		return runJournalShow(nil)
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "write", "add":
		return runJournalWrite(rest)
	case "mood":
		return runJournalMood(rest)
	case "show":
		return runJournalShow(rest)
	case "log":
		return runJournalLog(rest)
	}
	return fmt.Errorf("unknown journal subcommand %q", cmd)
}

func runJournalWrite(args []string) error {
	fs := flag.NewFlagSet("journal write", flag.ContinueOnError)
	body := fs.String("b", "", "entry body")
	args = reorderArgs(args, []string{"b"})
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *body == "" && fs.NArg() > 0 {
		*body = strings.Join(fs.Args(), " ")
	}
	if *body == "" {
		if data, err := readStdinIfPiped(); err == nil && data != "" {
			*body = data
		}
	}
	if *body == "" {
		return errors.New("body required (-b ... or stdin)")
	}
	st, err := open()
	if err != nil {
		return err
	}
	if err := journal.Append(st, time.Now(), *body); err != nil {
		return err
	}
	fmt.Println(ui.Green("journal updated"))
	return nil
}

func runJournalMood(args []string) error {
	fs := flag.NewFlagSet("journal mood", flag.ContinueOnError)
	mood := fs.Int("mood", 0, "1..5")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *mood == 0 && fs.NArg() > 0 {
		v, err := strconv.Atoi(fs.Arg(0))
		if err != nil {
			return err
		}
		*mood = v
	}
	st, err := open()
	if err != nil {
		return err
	}
	if err := journal.SetMood(st, time.Now(), *mood); err != nil {
		return err
	}
	fmt.Println(ui.Green("mood recorded"))
	return nil
}

func runJournalShow(args []string) error {
	fs := flag.NewFlagSet("journal show", flag.ContinueOnError)
	dateStr := fs.String("date", "", "YYYY-MM-DD")
	if err := fs.Parse(args); err != nil {
		return err
	}
	day := time.Now()
	if *dateStr != "" {
		t, err := time.Parse("2006-01-02", *dateStr)
		if err != nil {
			return err
		}
		day = t
	}
	st, err := open()
	if err != nil {
		return err
	}
	e, ok := journal.Get(st, day)
	if !ok {
		fmt.Println(ui.Dim("(no entry for "+day.Format("2006-01-02")+")"))
		return nil
	}
	ui.PrintHeader(os.Stdout, "Journal — "+e.Date)
	if e.Mood > 0 {
		fmt.Println(ui.Yellow("mood: "+moodGlyph(e.Mood)), "(", e.Mood, "/5)")
	}
	for _, sec := range e.Sections {
		fmt.Println()
		fmt.Println(ui.Dim(sec.At.Local().Format("15:04")))
		fmt.Println(sec.Body)
	}
	return nil
}

func runJournalLog(args []string) error {
	fs := flag.NewFlagSet("journal log", flag.ContinueOnError)
	days := fs.Int("days", 14, "how many days back")
	if err := fs.Parse(args); err != nil {
		return err
	}
	st, err := open()
	if err != nil {
		return err
	}
	all := journal.All(st)
	cutoff := time.Now().AddDate(0, 0, -*days).Format("2006-01-02")
	for _, e := range all {
		if e.Date < cutoff {
			continue
		}
		ui.PrintHeader(os.Stdout, e.Date)
		if e.Mood > 0 {
			fmt.Println(ui.Yellow("mood: "+moodGlyph(e.Mood)))
		}
		for _, sec := range e.Sections {
			fmt.Println(ui.Dim(sec.At.Local().Format("15:04")))
			fmt.Println(sec.Body)
			fmt.Println()
		}
	}
	return nil
}

// ----------------------------------------------------------------------------
// pomodoro
// ----------------------------------------------------------------------------

func runPomodoro(args []string) error {
	fs := flag.NewFlagSet("pomodoro", flag.ContinueOnError)
	label := fs.String("label", "", "what are you working on?")
	rounds := fs.Int("rounds", 1, "how many focus rounds")
	focus := fs.Int("focus", 0, "minutes per focus phase")
	short := fs.Int("short-break", 0, "minutes per short break")
	long := fs.Int("long-break", 0, "minutes per long break")
	until := fs.Int("until-long", 0, "focus rounds before long break")
	quiet := fs.Bool("quiet", false, "suppress beeps")
	args = reorderArgs(args, []string{"label", "rounds", "focus", "short-break", "long-break", "until-long"})
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *label == "" && fs.NArg() > 0 {
		*label = strings.Join(fs.Args(), " ")
	}
	st, err := open()
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	return pomodoro.Run(ctx, os.Stdout, st, pomodoro.Options{
		Label:        *label,
		Rounds:       *rounds,
		FocusMinutes: *focus,
		ShortBreak:   *short,
		LongBreak:    *long,
		UntilLong:    *until,
		Quiet:        *quiet,
	})
}

// ----------------------------------------------------------------------------
// stats / today / tags / quote / etc.
// ----------------------------------------------------------------------------

func runStats(args []string) error {
	st, err := open()
	if err != nil {
		return err
	}
	d, err := st.Snapshot()
	if err != nil {
		return err
	}
	now := time.Now()
	s := stats.Compute(d, now)
	ui.PrintHeader(os.Stdout, "MindForge — stats")
	fmt.Printf("  notes:       %s\n", ui.Bold(itoa(s.Notes)))
	fmt.Printf("  tasks:       %s open · %s overdue · %s total\n",
		ui.Bold(itoa(s.OpenTasks)),
		ui.Red(itoa(s.Overdue)),
		ui.Dim(itoa(s.Tasks)))
	fmt.Printf("  journal:     %s days · %s day streak\n",
		ui.Bold(itoa(s.JournalDays)),
		streakBadge(s.JournalStreak))
	fmt.Printf("  pomodoros:   %s today · %s this week · %s total\n",
		ui.Bold(itoa(s.PomodorosToday)),
		ui.Bold(itoa(s.PomodorosThisWeek)),
		ui.Dim(itoa(s.PomodorosTotal)))
	fmt.Printf("  focus time:  %s today · %s lifetime\n",
		ui.Bold(formatMinutes(s.FocusMinutesToday)),
		ui.Dim(formatMinutes(s.FocusMinutesTotal)))
	fmt.Println()
	fmt.Println(ui.Bold("Last 7 weeks of pomodoros"))
	weekdayLabels := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	for wd := 1; wd <= 7; wd++ {
		fmt.Printf("  %s ", weekdayLabels[wd-1])
		for w := 6; w >= 0; w-- {
			day := wd % 7
			fmt.Print(heatCell(s.HeatLastWeeks[w][day]))
		}
		fmt.Println()
	}
	fmt.Println(ui.Dim("  ░ none  ▒ 1-2  ▓ 3-4  █ 5+"))
	fmt.Println()
	tt := stats.TopTags(d, 5)
	if len(tt) > 0 {
		fmt.Println(ui.Bold("Top tags"))
		for _, t := range tt {
			fmt.Printf("  %s  %s\n", ui.Pad("#"+t.Tag, 16), ui.Dim(itoa(t.Count)+" item(s)"))
		}
	}
	return nil
}

func runToday(w io.Writer) error {
	st, err := open()
	if err != nil {
		return err
	}
	d, err := st.Snapshot()
	if err != nil {
		return err
	}
	now := time.Now()
	ui.PrintHeader(w, "Today — "+now.Format("Mon 02 Jan 2006"))

	fmt.Fprintln(w, ui.Bold("Pinned notes"))
	any := false
	for _, n := range d.Notes {
		if !n.Pinned {
			continue
		}
		any = true
		fmt.Fprintf(w, "  %s  %s %s\n", ui.Yellow("*"), "#"+itoa(n.ID), n.Title)
	}
	if !any {
		fmt.Fprintln(w, ui.Dim("  (none)"))
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, ui.Bold("Top open tasks"))
	open := tasks.List(st, tasks.FilterOpts{})
	if len(open) == 0 {
		fmt.Fprintln(w, ui.Dim("  (no open tasks — nice)"))
	}
	for i, t := range open {
		if i >= 7 {
			break
		}
		due := ""
		if t.Due != nil {
			due = " · " + formatDueShort(*t.Due, now)
		}
		fmt.Fprintf(w, "  %s %s %s%s\n",
			ui.Bold("P"+itoa(t.Priority)),
			ui.Dim("#"+itoa(t.ID)),
			t.Title,
			ui.Yellow(due))
	}

	if e, ok := journal.Get(st, now); ok && len(e.Sections) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, ui.Bold("Journal — already today"))
		for _, sec := range e.Sections {
			fmt.Fprintf(w, "  %s %s\n", ui.Dim(sec.At.Local().Format("15:04")), truncate(sec.Body, 80))
		}
	} else {
		fmt.Fprintln(w)
		fmt.Fprintln(w, ui.Dim("Journal: empty for today.  Try `mindforge journal write -b \"...\"`."))
	}

	s := stats.Compute(d, now)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s focus today · %s pomodoros today · %s day streak\n",
		ui.Bold(formatMinutes(s.FocusMinutesToday)),
		ui.Bold(itoa(s.PomodorosToday)),
		streakBadge(s.JournalStreak))

	if !d.Settings.DisableMotivation {
		fmt.Fprintln(w)
		fmt.Fprintln(w, ui.Italic(ui.Magenta(randomQuote())))
	}
	return nil
}

func runTags(args []string) error {
	st, err := open()
	if err != nil {
		return err
	}
	cnt := notes.AllTags(st)
	d, err := st.Snapshot()
	if err != nil {
		return err
	}
	for _, t := range d.Tasks {
		for _, tag := range t.Tags {
			cnt[tag]++
		}
	}
	if len(cnt) == 0 {
		fmt.Println(ui.Dim("(no tags)"))
		return nil
	}
	keys := make([]string, 0, len(cnt))
	for k := range cnt {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if cnt[keys[i]] != cnt[keys[j]] {
			return cnt[keys[i]] > cnt[keys[j]]
		}
		return keys[i] < keys[j]
	})
	rows := make([][]string, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, []string{"#" + k, itoa(cnt[k])})
	}
	ui.PrintTable(os.Stdout, []string{"tag", "count"}, rows)
	return nil
}

func runWhere(args []string) error {
	st, err := open()
	if err != nil {
		return err
	}
	fmt.Println(st.Path())
	return nil
}

func runBackup(args []string) error {
	st, err := open()
	if err != nil {
		return err
	}
	dst, err := st.Backup()
	if err != nil {
		return err
	}
	fmt.Println(ui.Green("backup:"), dst)
	return nil
}

func runExport(args []string) error {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	format := fs.String("format", "md", "json | md | csv")
	out := fs.String("out", "", "write to file (defaults to stdout)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	st, err := open()
	if err != nil {
		return err
	}
	d, err := st.Snapshot()
	if err != nil {
		return err
	}
	var w io.Writer = os.Stdout
	if *out != "" {
		f, err := os.Create(*out)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}
	switch *format {
	case "json":
		return writeJSON(w, d)
	case "md", "markdown":
		return writeMarkdown(w, d)
	case "csv":
		return writeCSV(w, d)
	}
	return fmt.Errorf("unknown format %q", *format)
}

func runConfig(args []string) error {
	fs := flag.NewFlagSet("config", flag.ContinueOnError)
	get := fs.String("get", "", "key to read")
	set := fs.String("set", "", "key=value to write")
	if err := fs.Parse(args); err != nil {
		return err
	}
	st, err := open()
	if err != nil {
		return err
	}
	if *get != "" {
		val, err := configGet(st, *get)
		if err != nil {
			return err
		}
		fmt.Println(val)
		return nil
	}
	if *set != "" {
		parts := strings.SplitN(*set, "=", 2)
		if len(parts) != 2 {
			return errors.New("--set requires key=value")
		}
		return configSet(st, parts[0], parts[1])
	}
	// no flags — show all
	d, err := st.Snapshot()
	if err != nil {
		return err
	}
	fmt.Printf("pomodoro_minutes      = %d\n", d.Settings.PomodoroMinutes)
	fmt.Printf("short_break_minutes   = %d\n", d.Settings.ShortBreakMinutes)
	fmt.Printf("long_break_minutes    = %d\n", d.Settings.LongBreakMinutes)
	fmt.Printf("pomodoros_until_long  = %d\n", d.Settings.PomodorosUntilLong)
	fmt.Printf("date_format           = %s\n", d.Settings.DateFormat)
	fmt.Printf("disable_motivation    = %t\n", d.Settings.DisableMotivation)
	fmt.Printf("disable_backup_on_save= %t\n", d.Settings.DisableBackupOnSave)
	return nil
}

func runDoctor(w io.Writer) error {
	st, err := open()
	if err != nil {
		fmt.Fprintln(w, ui.Red("FAIL"), err)
		return err
	}
	fmt.Fprintln(w, ui.Green("OK"), "data file readable at", st.Path())
	d, err := st.Snapshot()
	if err != nil {
		fmt.Fprintln(w, ui.Red("FAIL"), "snapshot:", err)
		return err
	}
	if d.SchemaVersion != store.SchemaVersion {
		fmt.Fprintln(w, ui.Yellow("WARN"), "schema version is", d.SchemaVersion, "expected", store.SchemaVersion)
	} else {
		fmt.Fprintln(w, ui.Green("OK"), "schema version", d.SchemaVersion)
	}
	fmt.Fprintln(w, ui.Green("OK"),
		fmt.Sprintf("%d notes, %d tasks, %d journal entries, %d pomodoros",
			len(d.Notes), len(d.Tasks), len(d.Journal), len(d.Pomodoros)))
	return nil
}

func runQuote(w io.Writer) error {
	fmt.Fprintln(w, ui.Italic(ui.Magenta(randomQuote())))
	return nil
}

// ----------------------------------------------------------------------------
// helpers
// ----------------------------------------------------------------------------

func itoa(n int) string { return strconv.Itoa(n) }

// reorderArgs moves all flag-like args (and their values) before any
// positional args so that flag.Parse can succeed even when the user
// writes the natural `command "title" --flag value` form.  valueFlags
// is the set of flags that take a separate value rather than being
// boolean toggles.
func reorderArgs(args []string, valueFlags []string) []string {
	value := map[string]bool{}
	for _, f := range valueFlags {
		value[f] = true
	}
	var flagArgs, positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			// Everything past `--` is positional.
			positional = append(positional, args[i+1:]...)
			break
		}
		if strings.HasPrefix(a, "-") && len(a) > 1 {
			flagArgs = append(flagArgs, a)
			base := strings.TrimLeft(a, "-")
			base = strings.SplitN(base, "=", 2)[0]
			if value[base] && !strings.Contains(a, "=") && i+1 < len(args) {
				flagArgs = append(flagArgs, args[i+1])
				i++
			}
			continue
		}
		positional = append(positional, a)
	}
	return append(flagArgs, positional...)
}

func splitTags(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.TrimPrefix(p, "#")
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func renderTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	parts := make([]string, len(tags))
	for i, t := range tags {
		parts[i] = ui.Cyan("#" + t)
	}
	return strings.Join(parts, " ")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

func parseDateMaybe(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	now := time.Now()
	switch s {
	case "today":
		t := startOfDay(now).Add(20 * time.Hour)
		return &t, nil
	case "tomorrow":
		t := startOfDay(now).AddDate(0, 0, 1).Add(20 * time.Hour)
		return &t, nil
	}
	if strings.HasPrefix(s, "+") && strings.HasSuffix(s, "d") {
		n, err := strconv.Atoi(s[1 : len(s)-1])
		if err == nil {
			t := startOfDay(now).AddDate(0, 0, n).Add(20 * time.Hour)
			return &t, nil
		}
	}
	formats := []string{
		"2006-01-02 15:04",
		"2006-01-02T15:04",
		"2006-01-02",
		"01/02 15:04",
		"01/02",
	}
	for _, f := range formats {
		if t, err := time.ParseInLocation(f, s, time.Local); err == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("cannot parse date %q", s)
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func readStdinIfPiped() (string, error) {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return "", err
	}
	if (fi.Mode() & os.ModeCharDevice) != 0 {
		return "", nil
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(data), "\r\n"), nil
}

func formatDueShort(due, now time.Time) string {
	d := startOfDay(due).Sub(startOfDay(now))
	days := int(d.Hours() / 24)
	switch {
	case days == 0:
		return "today"
	case days == 1:
		return "tomorrow"
	case days == -1:
		return "yesterday"
	case days < 0:
		return ui.Red(strconv.Itoa(-days) + "d ago")
	default:
		return strconv.Itoa(days) + "d"
	}
}

func formatMinutes(m int) string {
	if m < 60 {
		return strconv.Itoa(m) + "m"
	}
	return fmt.Sprintf("%dh %02dm", m/60, m%60)
}

func streakBadge(n int) string {
	if n == 0 {
		return ui.Dim("0")
	}
	if n >= 7 {
		return ui.Green(strconv.Itoa(n) + " (on fire)")
	}
	return ui.Yellow(strconv.Itoa(n))
}

func heatCell(n int) string {
	switch {
	case n == 0:
		return ui.Dim("░")
	case n <= 2:
		return ui.Cyan("▒")
	case n <= 4:
		return ui.Magenta("▓")
	default:
		return ui.Green("█")
	}
}

func moodGlyph(m int) string {
	switch m {
	case 1:
		return ":("
	case 2:
		return ":/"
	case 3:
		return ":|"
	case 4:
		return ":)"
	case 5:
		return ":D"
	}
	return "?"
}

func strikethrough(s string) string {
	if !ui.Enabled() {
		return s
	}
	return "\x1b[9m" + s + "\x1b[29m"
}

// configGet/Set are keyed by the JSON tag of Settings.
func configGet(st *store.Store, key string) (string, error) {
	d, err := st.Snapshot()
	if err != nil {
		return "", err
	}
	switch key {
	case "pomodoro_minutes":
		return itoa(d.Settings.PomodoroMinutes), nil
	case "short_break_minutes":
		return itoa(d.Settings.ShortBreakMinutes), nil
	case "long_break_minutes":
		return itoa(d.Settings.LongBreakMinutes), nil
	case "pomodoros_until_long":
		return itoa(d.Settings.PomodorosUntilLong), nil
	case "date_format":
		return d.Settings.DateFormat, nil
	case "disable_motivation":
		return strconv.FormatBool(d.Settings.DisableMotivation), nil
	case "disable_backup_on_save":
		return strconv.FormatBool(d.Settings.DisableBackupOnSave), nil
	}
	return "", fmt.Errorf("unknown key %q", key)
}

func configSet(st *store.Store, key, value string) error {
	return st.Use(func(d *store.Data) error {
		intVal, intErr := strconv.Atoi(value)
		boolVal, _ := strconv.ParseBool(value)
		switch key {
		case "pomodoro_minutes":
			if intErr != nil {
				return intErr
			}
			d.Settings.PomodoroMinutes = intVal
		case "short_break_minutes":
			if intErr != nil {
				return intErr
			}
			d.Settings.ShortBreakMinutes = intVal
		case "long_break_minutes":
			if intErr != nil {
				return intErr
			}
			d.Settings.LongBreakMinutes = intVal
		case "pomodoros_until_long":
			if intErr != nil {
				return intErr
			}
			d.Settings.PomodorosUntilLong = intVal
		case "date_format":
			d.Settings.DateFormat = value
		case "disable_motivation":
			d.Settings.DisableMotivation = boolVal
		case "disable_backup_on_save":
			d.Settings.DisableBackupOnSave = boolVal
		default:
			return fmt.Errorf("unknown key %q", key)
		}
		return nil
	})
}

// Surface filepath for use by tests.
var _ = filepath.Join
