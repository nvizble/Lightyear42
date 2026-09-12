package notify

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileStoreLoadMissingFile(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "sub", "notifications.json"))

	state, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if state.Seeded {
		t.Error("Seeded = true, want false for a missing file")
	}
	if len(state.Evaluations) != 0 {
		t.Errorf("Evaluations = %v, want empty", state.Evaluations)
	}
}

func TestFileStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "notifications.json")
	store := NewFileStore(path)

	begin := time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC)
	if err := store.Save(State{Evaluations: map[int]time.Time{42: begin}}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	state, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !state.Seeded {
		t.Error("Seeded = false, want true after a save")
	}
	if !state.Has(42) {
		t.Fatalf("Has(42) = false, state = %v", state.Evaluations)
	}
	if got := state.Evaluations[42]; !got.Equal(begin) {
		t.Errorf("begin = %v, want %v", got, begin)
	}
}

func TestFileStoreSaveIsPrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notifications.json")
	if err := NewFileStore(path).Save(State{}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("perm = %o, want 600", perm)
	}
}

func TestFileStoreSaveEmptyStateIsSeeded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notifications.json")
	store := NewFileStore(path)

	// A user with no evaluations must still get a baseline, otherwise the
	// first real booking would be swallowed as "seeding".
	if err := store.Save(State{}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	state, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !state.Seeded {
		t.Error("Seeded = false, want true")
	}
	if state.Evaluations == nil {
		t.Error("Evaluations = nil, want empty map")
	}
}

func TestFileStoreLoadCorrupted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notifications.json")
	if err := os.WriteFile(path, []byte("{não é json"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := NewFileStore(path).Load(); err == nil {
		t.Fatal("Load() = nil error, want failure on corrupted state")
	}
}

func TestFileStoreSaveOverwrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notifications.json")
	store := NewFileStore(path)

	if err := store.Save(State{Evaluations: map[int]time.Time{1: time.Now()}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := store.Save(State{Evaluations: map[int]time.Time{2: time.Now()}}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	state, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if state.Has(1) {
		t.Error("Has(1) = true, want the old entry to be gone")
	}
	if !state.Has(2) {
		t.Error("Has(2) = false, want the new entry")
	}

	// The atomic write must not leave temp files behind.
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("dir has %d entries, want only the state file", len(entries))
	}
}
