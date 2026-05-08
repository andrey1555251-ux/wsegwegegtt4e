// MindForge — a personal productivity companion.
//
// This is the very first program I would write if I were a person:
// a small, friendly CLI tool that helps you keep track of your
// thoughts, tasks, and time. It uses only the Go standard library
// so the resulting binary is a single self-contained .exe.
package main

import (
	"fmt"
	"os"

	"github.com/andrey1555251-ux/mindforge/internal/cli"
	"github.com/andrey1555251-ux/mindforge/internal/ui"
)

func main() {
	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, ui.Red("error: ")+err.Error())
		os.Exit(1)
	}
}
