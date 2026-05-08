// Package store handles persistence to a single JSON file under the
// user's config directory.  Everything MindForge knows about you lives
// in one place so backing up your data is just `cp` of one file.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SchemaVersion bumps when the on-disk layout changes in a backwards
// incompatible way.  See migrate() for the upgrade path.
const SchemaVersion = 2

// Note is a free-form text snippet with tags.
type Note struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Tags      []string  `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Pinned    bool      `json:"pinned,omitempty"`
}

// Task represents a TODO entry.
type Task struct {
	ID        int        `json:"id"`
	Title     string     `json:"title"`
	Notes     string     `json:"notes,omitempty"`
	Priority  int        `json:"priority"` // 1 (highest) .. 5 (lowest)
	Due       *time.Time `json:"due,omitempty"`
	Done      bool       `json:"done"`
	DoneAt    *time.Time `json:"done_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	Tags      []string   `json:"tags,omitempty"`
}

// JournalEntry is one dated reflection.  We allow at most one entry per
// day on creation but appends are recorded as separate Sections so a day
// can grow incrementally.
type JournalEntry struct {
	Date     string    `json:"date"` // YYYY-MM-DD
	Sections []Section `json:"sections"`
	Mood     int       `json:"mood,omitempty"` // 1..5
}

type Section struct {
	At   time.Time `json:"at"`
	Body string    `json:"body"`
}

// PomodoroLog records one completed pomodoro for the streak and stats.
type PomodoroLog struct {
	StartedAt   time.Time `json:"started_at"`
	FinishedAt  time.Time `json:"finished_at"`
	Minutes     int       `json:"minutes"`
	Label       string    `json:"label,omitempty"`
	Interrupted bool      `json:"interrupted,omitempty"`
}

// Data is the root JSON document persisted to disk.
type Data struct {
	SchemaVersion int            `json:"schema_version"`
	CreatedAt     time.Time      `json:"created_at"`
	Notes         []Note         `json:"notes"`
	Tasks         []Task         `json:"tasks"`
	Journal       []JournalEntry `json:"journal"`
	Pomodoros     []PomodoroLog  `json:"pomodoros"`
	Counters      Counters       `json:"counters"`
	Settings      Settings       `json:"settings"`
}

// Counters keep the next id for each kind.  We never reuse an id
// even after deletion so external references stay valid.
type Counters struct {
	NextNoteID int `json:"next_note_id"`
	NextTaskID int `json:"next_task_id"`
}

// Settings is for user preferences.
type Settings struct {
	PomodoroMinutes      int    `json:"pomodoro_minutes"`
	ShortBreakMinutes    int    `json:"short_break_minutes"`
	LongBreakMinutes     int    `json:"long_break_minutes"`
	PomodorosUntilLong   int    `json:"pomodoros_until_long"`
	DefaultEditor        string `json:"default_editor,omitempty"`
	DateFormat           string `json:"date_format,omitempty"`
	DisableMotivation    bool   `json:"disable_motivation,omitempty"`
	DisableBackupOnSave  bool   `json:"disable_backup_on_save,omitempty"`
}

// Store wraps Data with locking and persistence.
type Store struct {
	mu   sync.Mutex
	path string
	data *Data
}

// Default returns the canonical place we keep the data file.
// On Windows that's %APPDATA%\mindforge\data.json, on macOS/Linux it
// falls under XDG_CONFIG_HOME or ~/.config/mindforge/data.json.
func Default() (*Store, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return Open(filepath.Join(dir, "data.json"))
}

func configDir() (string, error) {
	if dir := os.Getenv("MINDFORGE_HOME"); dir != "" {
		return dir, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "mindforge"), nil
}

// Open loads (or creates) a data file at path.
func Open(path string) (*Store, error) {
	s := &Store{path: path}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// Path returns the path of the underlying data file.
func (s *Store) Path() string { return s.path }

func defaultData() *Data {
	return &Data{
		SchemaVersion: SchemaVersion,
		CreatedAt:     time.Now(),
		Notes:         []Note{},
		Tasks:         []Task{},
		Journal:       []JournalEntry{},
		Pomodoros:     []PomodoroLog{},
		Counters:      Counters{NextNoteID: 1, NextTaskID: 1},
		Settings: Settings{
			PomodoroMinutes:    25,
			ShortBreakMinutes:  5,
			LongBreakMinutes:   15,
			PomodorosUntilLong: 4,
			DateFormat:         "2006-01-02",
		},
	}
}

func (s *Store) load() error {
	f, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) {
		s.data = defaultData()
		return s.save()
	}
	if err != nil {
		return err
	}
	defer f.Close()

	raw, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	if len(raw) == 0 {
		s.data = defaultData()
		return s.save()
	}
	d := &Data{}
	if err := json.Unmarshal(raw, d); err != nil {
		return fmt.Errorf("parsing %s: %w", s.path, err)
	}
	if err := migrate(d); err != nil {
		return err
	}
	s.data = d
	return nil
}

func migrate(d *Data) error {
	if d.SchemaVersion == 0 {
		d.SchemaVersion = 1
	}
	// v1 -> v2: settings block was added; fill in defaults if missing.
	if d.Settings.PomodoroMinutes == 0 {
		d.Settings = defaultData().Settings
	}
	if d.Settings.DateFormat == "" {
		d.Settings.DateFormat = "2006-01-02"
	}
	if d.Counters.NextNoteID == 0 {
		d.Counters.NextNoteID = 1
		for _, n := range d.Notes {
			if n.ID >= d.Counters.NextNoteID {
				d.Counters.NextNoteID = n.ID + 1
			}
		}
	}
	if d.Counters.NextTaskID == 0 {
		d.Counters.NextTaskID = 1
		for _, t := range d.Tasks {
			if t.ID >= d.Counters.NextTaskID {
				d.Counters.NextTaskID = t.ID + 1
			}
		}
	}
	d.SchemaVersion = SchemaVersion
	return nil
}

// save atomically writes the current data to disk.  We write to a sibling
// .tmp file first and rename — that way a power loss mid-write cannot
// produce a half-written JSON file.
func (s *Store) save() error {
	if s.data == nil {
		return errors.New("nil data")
	}
	tmp := s.path + ".tmp"
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return err
	}
	return nil
}

// Use runs fn under the store's lock and persists the change unless
// fn returns an error.
func (s *Store) Use(fn func(*Data) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := fn(s.data); err != nil {
		return err
	}
	return s.save()
}

// View runs fn under the store's lock without persisting.
func (s *Store) View(fn func(*Data)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(s.data)
}

// Snapshot returns a deep-ish copy of the data for read-only use.  We
// rely on json (de)serialization to copy nested slices safely.
func (s *Store) Snapshot() (*Data, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := json.Marshal(s.data)
	if err != nil {
		return nil, err
	}
	out := &Data{}
	if err := json.Unmarshal(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Backup writes a timestamped copy of the data file next to it and
// returns the full path of the new file.
func (s *Store) Backup() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stamp := time.Now().Format("20060102-150405")
	dst := s.path + "." + stamp + ".bak"
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(dst, raw, 0o644); err != nil {
		return "", err
	}
	return dst, nil
}
