// Package ui contains tiny presentation helpers: ANSI colors,
// a banner, table formatting and tiny progress widgets.
//
// The colors auto-disable themselves when stdout is not a terminal,
// when NO_COLOR is set, or on legacy Windows shells where ANSI
// escapes are not honoured.
package ui

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"unicode/utf8"
)

const (
	reset     = "\x1b[0m"
	boldSeq   = "\x1b[1m"
	dimSeq    = "\x1b[2m"
	italicSeq = "\x1b[3m"
	underline = "\x1b[4m"
)

// Color codes are kept simple on purpose.
var (
	red     = "\x1b[31m"
	green   = "\x1b[32m"
	yellow  = "\x1b[33m"
	blue    = "\x1b[34m"
	magenta = "\x1b[35m"
	cyan    = "\x1b[36m"
	gray    = "\x1b[90m"
)

var enabled = computeEnabled()

func computeEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("MINDFORGE_NO_COLOR") != "" {
		return false
	}
	// Best effort: assume modern Windows 10+ terminals support ANSI.
	// Users on legacy cmd.exe can opt out with NO_COLOR=1.
	if runtime.GOOS == "windows" && os.Getenv("WT_SESSION") == "" && os.Getenv("TERM") == "" {
		// We still allow colors in modern Windows Terminals (which set WT_SESSION).
		// On classic cmd.exe ANSI is silently swallowed; ENABLE_VIRTUAL_TERMINAL_PROCESSING
		// is enabled automatically by Windows 10+ build 14931.  We err on the side
		// of being colorful — worst case the user sees escape sequences and disables
		// them with NO_COLOR=1.
		return true
	}
	return true
}

// SetEnabled forces color output on or off.
func SetEnabled(on bool) { enabled = on }

// Enabled reports whether color output is currently active.
func Enabled() bool { return enabled }

func wrap(code, s string) string {
	if !enabled {
		return s
	}
	return code + s + reset
}

// Color helpers.
func Red(s string) string     { return wrap(red, s) }
func Green(s string) string   { return wrap(green, s) }
func Yellow(s string) string  { return wrap(yellow, s) }
func Blue(s string) string    { return wrap(blue, s) }
func Magenta(s string) string { return wrap(magenta, s) }
func Cyan(s string) string    { return wrap(cyan, s) }
func Gray(s string) string    { return wrap(gray, s) }
func Bold(s string) string    { return wrap(boldSeq, s) }
func Dim(s string) string     { return wrap(dimSeq, s) }
func Italic(s string) string  { return wrap(italicSeq, s) }
func Under(s string) string   { return wrap(underline, s) }

// Banner returns the welcome banner shown by `mindforge` with no args.
// It is intentionally hand-drawn ASCII so it works even with a 1-bit
// font on the most ascetic Windows console.
func Banner(version string) string {
	const art = `
  __  __ _           _ ____                       
 |  \/  (_)_ __   __| |  _ \ ___  _ __ __ _  ___ 
 | |\/| | | '_ \ / _' | |_) / _ \| '__/ _' |/ _ \
 | |  | | | | | | (_| |  __/ (_) | | | (_| |  __/
 |_|  |_|_|_| |_|\__,_|_|   \___/|_|  \__, |\___|
                                      |___/      
`
	return Cyan(art) + "  " + Dim("a tiny productivity companion · v"+version) + "\n"
}

// PrintHeader writes a single underlined heading to w.
func PrintHeader(w io.Writer, title string) {
	fmt.Fprintln(w, Bold(Cyan(title)))
	fmt.Fprintln(w, Dim(strings.Repeat("─", visibleWidth(title))))
}

// visibleWidth returns the number of *visible* runes in s,
// approximating the terminal width of the rendered string.
// We strip ANSI escapes here too so the helper works after coloring.
func visibleWidth(s string) int {
	stripped := stripANSI(s)
	return utf8.RuneCountInString(stripped)
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
				j++
			}
			i = j
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// Pad pads a (possibly colored) string to width w using right padding.
func Pad(s string, w int) string {
	diff := w - visibleWidth(s)
	if diff <= 0 {
		return s
	}
	return s + strings.Repeat(" ", diff)
}

// PrintTable renders a small ASCII table.  It is good enough for
// listing < a few hundred rows and avoids pulling in any third-party
// package so the binary stays minimal.
func PrintTable(w io.Writer, headers []string, rows [][]string) {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = visibleWidth(h)
	}
	for _, row := range rows {
		for i, c := range row {
			if i >= len(widths) {
				continue
			}
			if v := visibleWidth(c); v > widths[i] {
				widths[i] = v
			}
		}
	}

	for i, h := range headers {
		fmt.Fprint(w, Bold(Pad(h, widths[i])))
		if i < len(headers)-1 {
			fmt.Fprint(w, "  ")
		}
	}
	fmt.Fprintln(w)
	for i := range headers {
		fmt.Fprint(w, Dim(strings.Repeat("─", widths[i])))
		if i < len(headers)-1 {
			fmt.Fprint(w, "  ")
		}
	}
	fmt.Fprintln(w)

	for _, row := range rows {
		for i, c := range row {
			if i >= len(widths) {
				continue
			}
			fmt.Fprint(w, Pad(c, widths[i]))
			if i < len(row)-1 {
				fmt.Fprint(w, "  ")
			}
		}
		fmt.Fprintln(w)
	}
}

// Progress draws a [#####-----] style bar of width n filled to fraction f.
func Progress(n int, f float64) string {
	if n <= 0 {
		return ""
	}
	if f < 0 {
		f = 0
	}
	if f > 1 {
		f = 1
	}
	full := int(float64(n) * f)
	return "[" + strings.Repeat("#", full) + strings.Repeat("-", n-full) + "]"
}
