package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/andrey1555251-ux/mindforge/internal/habits"
	"github.com/andrey1555251-ux/mindforge/internal/store"
	"github.com/andrey1555251-ux/mindforge/internal/ui"
)

// runHabit dispatches `mindforge habit <sub>`.  When called bare it
// shows the same view as `habit list` so a typo-free curious user can
// always discover what's available.
func runHabit(args []string) error {
	if len(args) == 0 {
		return runHabitList(nil)
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "add", "new":
		return runHabitAdd(rest)
	case "list", "ls":
		return runHabitList(rest)
	case "check", "do", "tick":
		return runHabitMark(rest, true)
	case "uncheck", "undo", "untick":
		return runHabitMark(rest, false)
	case "rename":
		return runHabitRename(rest)
	case "delete", "rm", "del":
		return runHabitDelete(rest)
	case "history", "log":
		return runHabitHistory(rest)
	}
	return fmt.Errorf("unknown habit subcommand %q", cmd)
}

func runHabitAdd(args []string) error {
	fs := flag.NewFlagSet("habit add", flag.ContinueOnError)
	target := fs.Int("target", 1, "daily target count (informational)")
	args = reorderArgs(args, []string{"target"})
	if err := fs.Parse(args); err != nil {
		return err
	}
	name := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if name == "" {
		return errors.New("usage: mindforge habit add \"<name>\"")
	}
	st, err := open()
	if err != nil {
		return err
	}
	id, err := habits.Add(st, name, *target)
	if err != nil {
		return err
	}
	fmt.Println(ui.Green("habit added"), ui.Dim("#"+itoa(id)))
	return nil
}

func runHabitList(args []string) error {
	fs := flag.NewFlagSet("habit list", flag.ContinueOnError)
	weeks := fs.Int("weeks", 2, "how many recent weeks to render")
	if err := fs.Parse(args); err != nil {
		return err
	}
	st, err := open()
	if err != nil {
		return err
	}
	all := habits.All(st)
	if len(all) == 0 {
		fmt.Println(ui.Dim("(no habits yet — try `mindforge habit add \"drink water\"`)"))
		return nil
	}
	now := time.Now()
	days := *weeks * 7
	if days < 7 {
		days = 7
	}

	// Print header row with abbreviated dates so the user can read the
	// rightmost column as today.
	header := []string{"id", "habit", "streak", "best", "today"}
	rows := make([][]string, 0, len(all))
	for _, h := range all {
		streak := habits.Streak(h, now)
		best := habits.Best(h)
		check := ui.Dim("·")
		if habits.CheckedOn(h, now) {
			check = ui.Green("v")
		}
		rows = append(rows, []string{
			"#" + itoa(h.ID),
			truncate(h.Name, 32),
			streakBadge(streak),
			ui.Dim(itoa(best)),
			check,
		})
	}
	ui.PrintTable(os.Stdout, header, rows)
	fmt.Println()
	fmt.Println(ui.Bold(fmt.Sprintf("Last %d days", days)))
	for _, h := range all {
		fmt.Printf("  %s %s ", ui.Pad("#"+itoa(h.ID), 4), ui.Pad(truncate(h.Name, 18), 18))
		for _, ok := range habits.LastNDays(h, now, days) {
			if ok {
				fmt.Print(ui.Green("█"))
			} else {
				fmt.Print(ui.Dim("░"))
			}
		}
		fmt.Println()
	}
	return nil
}

func runHabitMark(args []string, on bool) error {
	fs := flag.NewFlagSet("habit mark", flag.ContinueOnError)
	dateStr := fs.String("date", "", "date to mark (default today, YYYY-MM-DD or 'yesterday')")
	args = reorderArgs(args, []string{"date"})
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		if on {
			return errors.New("usage: mindforge habit check <id> [<id> ...]")
		}
		return errors.New("usage: mindforge habit uncheck <id> [<id> ...]")
	}
	day, err := parseHabitDate(*dateStr)
	if err != nil {
		return err
	}
	st, err := open()
	if err != nil {
		return err
	}
	for _, raw := range fs.Args() {
		id, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("invalid id %q", raw)
		}
		if on {
			if err := habits.Check(st, id, day); err != nil {
				return err
			}
			h, _ := habits.Get(st, id)
			fmt.Printf("%s %s · streak %s\n",
				ui.Green("checked"),
				ui.Dim("#"+itoa(id)+" "+h.Name),
				streakBadge(habits.Streak(h, time.Now())))
		} else {
			if err := habits.Uncheck(st, id, day); err != nil {
				return err
			}
			fmt.Println(ui.Yellow("unchecked"), ui.Dim("#"+itoa(id)))
		}
	}
	return nil
}

func runHabitRename(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: mindforge habit rename <id> \"new name\"")
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid id %q", args[0])
	}
	name := strings.Join(args[1:], " ")
	st, err := open()
	if err != nil {
		return err
	}
	if err := habits.Rename(st, id, name); err != nil {
		return err
	}
	fmt.Println(ui.Green("habit renamed"))
	return nil
}

func runHabitDelete(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: mindforge habit delete <id>")
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid id %q", args[0])
	}
	st, err := open()
	if err != nil {
		return err
	}
	if err := habits.Delete(st, id); err != nil {
		return err
	}
	fmt.Println(ui.Green("habit deleted"))
	return nil
}

func runHabitHistory(args []string) error {
	fs := flag.NewFlagSet("habit history", flag.ContinueOnError)
	weeks := fs.Int("weeks", 8, "how many weeks back to show")
	args = reorderArgs(args, []string{"weeks"})
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return errors.New("usage: mindforge habit history <id> [--weeks 8]")
	}
	id, err := strconv.Atoi(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("invalid id %q", fs.Arg(0))
	}
	st, err := open()
	if err != nil {
		return err
	}
	h, ok := habits.Get(st, id)
	if !ok {
		return fmt.Errorf("habit %d not found", id)
	}
	now := time.Now()
	ui.PrintHeader(os.Stdout, "Habit — "+h.Name)
	fmt.Printf("  current streak %s · best %s · %d total check-ins\n\n",
		streakBadge(habits.Streak(h, now)),
		ui.Bold(itoa(habits.Best(h))),
		len(h.Checks))
	days := *weeks * 7
	row := habits.LastNDays(h, now, days)
	weekdays := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	// Each column is one week; bottom row is most recent week.
	cols := *weeks
	for wd := 0; wd < 7; wd++ {
		fmt.Printf("  %s ", weekdays[wd])
		for c := cols - 1; c >= 0; c-- {
			idx := c*7 + wd
			if idx >= len(row) {
				fmt.Print(" ")
				continue
			}
			if row[idx] {
				fmt.Print(ui.Green("█"))
			} else {
				fmt.Print(ui.Dim("░"))
			}
		}
		fmt.Println()
	}
	return nil
}

func parseHabitDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Now(), nil
	}
	now := time.Now()
	switch s {
	case "today":
		return now, nil
	case "yesterday":
		return now.AddDate(0, 0, -1), nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

// EnsureHabitTable is a soft import to keep the linker quiet about the
// store dependency; we use store.Habit indirectly through the package.
var _ = store.Habit{}
