# MindForge

A tiny, single-binary productivity companion that lives in your terminal.

This was Devin's "first program if it were a person" — built end to end in
one session, only with the Go standard library, so the resulting binary
is a self-contained Windows `.exe` with no DLLs to chase.

It keeps these things in a single JSON file:

- **Notes** — quick markdown snippets with tags and pinning.
- **Tasks** — a TODO list with priorities, due dates and tags.
- **Journal** — append-only daily reflections with a 1‑5 mood.
- **Pomodoros** — a focus timer that records every completed round.
- **Habits** — daily check-ins with streaks and a heatmap.
- **Vault** — AES-256-GCM encrypted notes behind a passphrase.

…and turns those into a daily briefing, a tiny ASCII calendar, a
heatmap of focus time, full-text search, and Markdown / CSV / JSON
import & export.

## Quick start

```sh
mindforge welcome                    # banner + cheat sheet
mindforge add buy milk #home         # quick capture (creates a task)
mindforge add @ Idea: rewrite intro  # quick capture (creates a note)
mindforge task add "ship release" --priority 1 --due tomorrow --tag work
mindforge note add -t "first note" -b "this is the body" --tag ideas
mindforge journal write -b "today I shipped MindForge"
mindforge habit add "drink water"    # track a daily intention
mindforge habit check 1              # tick today's habit
mindforge pomodoro --label "deep work" --rounds 4
mindforge today                      # daily briefing
mindforge motd                       # one-liner banner for shell rc
mindforge stats                      # streak, heatmap, totals
mindforge search "release"           # search across notes, tasks, journal
mindforge vault encrypt 1            # AES-GCM encrypt a sensitive note
mindforge export --format md --out backup.md
```

Run `mindforge help` for the full menu.

## Where data lives

A single JSON file under your config directory:

| OS      | Default path                                     |
| ------- | ------------------------------------------------ |
| Windows | `%APPDATA%\mindforge\data.json`                  |
| Linux   | `$XDG_CONFIG_HOME/mindforge/data.json`            |
| macOS   | `~/Library/Application Support/mindforge/data.json` |

Override with `MINDFORGE_HOME=/some/path`.

## Building

You need Go 1.18+.

```sh
go build -o mindforge ./...                  # current platform
make windows                                 # cross-compile to .exe
make test                                    # run the unit tests
```

The `make windows` target produces `dist/mindforge.exe` for `amd64`
Windows.  The build is fully static — no CGO — so the resulting binary
runs on any Windows 10 / 11 host without extra runtime.

## Tour

```text
$ mindforge today

Today — Fri 08 May 2026
───────────────────────
Pinned notes
  *  #3 release notes draft

Top open tasks
  P1 #1 ship release · tomorrow
  P3 #4 fix sidebar width
  P3 #2 write blog post

Journal — empty for today.  Try `mindforge journal write -b "..."`.

25m focus today · 1 pomodoros today · 7 (on fire)

  start small.  finishing one thing beats starting ten.
```

```text
$ mindforge stats

MindForge — stats
─────────────────
  notes:       12
  tasks:       4 open · 1 overdue · 17 total
  journal:     14 days · 7 (on fire) day streak
  pomodoros:   1 today · 9 this week · 41 total
  focus time:  25m today · 17h 05m lifetime

Last 7 weeks of pomodoros
  Mon ░ ▒ ▓ █ ▓ ▒ ░
  Tue ░ ▒ ▒ ▓ ▒ ░ ░
  ...
```

## Why?

Most productivity apps want a server, an account, or a subscription.
MindForge is the opposite of that: one binary, one JSON file, no
network calls, no telemetry.  You can carry it on a USB stick.

## License

MIT
