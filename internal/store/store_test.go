package store

import (
	"path/filepath"
	"testing"
)

func TestOpenCreatesDataFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if s.Path() != path {
		t.Fatalf("path mismatch: %q vs %q", s.Path(), path)
	}
	d, err := s.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if d.SchemaVersion != SchemaVersion {
		t.Fatalf("schema mismatch: %d vs %d", d.SchemaVersion, SchemaVersion)
	}
	if d.Settings.PomodoroMinutes != 25 {
		t.Fatalf("default pomodoro: %d", d.Settings.PomodoroMinutes)
	}
}

func TestUseAndPersist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if err := s.Use(func(d *Data) error {
		d.Notes = append(d.Notes, Note{ID: 1, Title: "hello"})
		d.Counters.NextNoteID = 2
		return nil
	}); err != nil {
		t.Fatalf("use: %v", err)
	}

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	d, _ := s2.Snapshot()
	if len(d.Notes) != 1 || d.Notes[0].Title != "hello" {
		t.Fatalf("note not persisted: %+v", d.Notes)
	}
}

func TestBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	dst, err := s.Backup()
	if err != nil {
		t.Fatalf("backup: %v", err)
	}
	if dst == path {
		t.Fatalf("backup path equal to source")
	}
}

func TestMigrationFillsDefaults(t *testing.T) {
	d := &Data{}
	if err := migrate(d); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if d.Settings.PomodoroMinutes != 25 {
		t.Fatalf("expected default 25, got %d", d.Settings.PomodoroMinutes)
	}
	if d.Counters.NextNoteID == 0 {
		t.Fatalf("counter not bumped")
	}
}
