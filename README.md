# MindForge

A tiny, single-binary productivity companion that lives in your terminal.

This was Devin's "first program if it were a person" — built end to end in
one session, only with the Go standard library, so the resulting binary
is a self-contained Windows `.exe` with no DLLs to chase.

It keeps these things in a single JSON file:

- **Notes** — quick markdown snippets with tags and pinning.
- **Tasks** — TODO items with priorities, due dates, tags, recurrence, archiving.
- **Journal** — append-only daily reflections with a 1‑5 mood and a histogram view.
- **Pomodoros** — a focus timer that records every completed round.
- **Habits** — daily check-ins with streaks and a heatmap.
- **Vault** — AES-256-GCM encrypted notes behind a passphrase.

…and turns those into a daily briefing, a tiny ASCII calendar, an
agenda timeline, a markdown weekly review, full-text search, and
Markdown / CSV / JSON import & export.

## Quick start

```sh
mindforge welcome                    # banner + cheat sheet
mindforge add buy milk #home         # quick capture (creates a task)
mindforge add @ Idea: rewrite intro  # quick capture (creates a note)
mindforge task add "ship release" --priority 1 --due tomorrow --tag work
mindforge task next                  # one-line "what should I do now?" pointer
mindforge task pri 1 2               # change priority
mindforge task due 1 friday          # change due date (or 'none' to clear)
mindforge task repeat 1 weekly       # daily | weekly | monthly | none
mindforge note add -t "first note" -b "this is the body" --tag ideas
mindforge journal write -b "today I shipped MindForge"
mindforge journal mood 4             # 1..5
mindforge journal trend --days 30    # mood histogram
mindforge habit add "drink water"    # track a daily intention
mindforge habit check 1              # tick today's habit
mindforge pomodoro --label "deep work" --rounds 4
mindforge today                      # daily briefing
mindforge motd                       # one-liner banner for shell rc
mindforge agenda --days 7            # upcoming tasks grouped by day
mindforge summary --week             # markdown weekly review (paste-friendly)
mindforge stats                      # streak, heatmap, totals, focus by label
mindforge search "release"           # search across notes, tasks, journal
mindforge vault encrypt 1            # AES-GCM encrypt a sensitive note
mindforge import notes.md --kind notes
mindforge export --format md --out backup.md
mindforge doctor --fix               # self-check + auto-repair
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

Pending habits
  ○ #1 drink water  · 7 day streak

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

Top tags
  #work             12 item(s)
  #home              4 item(s)

Focus by label · last 30 days
  deep work               1h 25m · 4 rounds
  reading                   25m  · 1 rounds
```

```text
$ mindforge agenda

Agenda — next 7 days
────────────────────
OVERDUE
  P2  #7  reply to Anna  yesterday

Fri · 2026-05-08  (today)
  P1  #1  ship release  today  #work

Mon · 2026-05-11
  P3  #4  fix sidebar width  3d
```

```text
$ mindforge summary --week

# MindForge summary — last 7 days
_Fri 01 May → Fri 08 May 2026_

## ✓ Completed (3)
- [x] **P1** ship release  _#work_
- [x] **P3** fix sidebar width
- [x] **P3** write blog post

## ⏳ Open (4) · overdue (1)
- [ ] **P2** reply to Anna — _due 2026-05-07_

## 🍅 Focus
- 9 pomodoros · 3h 45m
  - **deep work** — 2h 30m
  - **reading** — 1h 15m

## 📓 Journal
- 5 days written
- avg mood: 3.80 / 5 _(across 5 entries)_
```

## Why?

Most productivity apps want a server, an account, or a subscription.
MindForge is the opposite of that: one binary, one JSON file, no
network calls, no telemetry.  You can carry it on a USB stick.

## License

MIT
