package notify

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// stateVersion guards the on-disk format against future changes.
const stateVersion = 1

// State is the record of evaluations the user was already notified about.
type State struct {
	// Seeded is false the very first time, when no state file exists yet.
	// Callers use it to record a baseline instead of notifying about every
	// evaluation already on the schedule.
	Seeded bool `json:"-"`
	// Evaluations maps a scale team id to its start time.
	Evaluations map[int]time.Time `json:"evaluations"`
}

// Has reports whether the user was already notified about this evaluation.
func (s State) Has(id int) bool {
	_, ok := s.Evaluations[id]
	return ok
}

// Store persists the notification state between runs.
type Store interface {
	Load() (State, error)
	Save(State) error
}

// FileStore keeps the state in a JSON file under the CLI data directory.
type FileStore struct {
	path string
}

// NewFileStore returns a store backed by the given file path.
func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

// stateFile is the JSON document written to disk.
type stateFile struct {
	Version     int               `json:"version"`
	Evaluations map[int]time.Time `json:"evaluations"`
}

// Load reads the state. A missing file is not an error: it yields an empty,
// unseeded state.
func (f *FileStore) Load() (State, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return State{Evaluations: map[int]time.Time{}}, nil
		}
		return State{}, fmt.Errorf("ler estado de notificações: %w", err)
	}

	var doc stateFile
	if err := json.Unmarshal(data, &doc); err != nil {
		return State{}, fmt.Errorf("estado de notificações corrompido (%s): %w", f.path, err)
	}
	if doc.Evaluations == nil {
		doc.Evaluations = map[int]time.Time{}
	}
	return State{Seeded: true, Evaluations: doc.Evaluations}, nil
}

// Save writes the state atomically, creating the directory when needed.
func (f *FileStore) Save(state State) error {
	dir := filepath.Dir(f.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("criar diretório de estado: %w", err)
	}

	evaluations := state.Evaluations
	if evaluations == nil {
		evaluations = map[int]time.Time{}
	}
	data, err := json.Marshal(stateFile{Version: stateVersion, Evaluations: evaluations})
	if err != nil {
		return fmt.Errorf("serializar estado de notificações: %w", err)
	}

	// Write to a sibling temp file and rename, so a crash mid-write never
	// leaves a truncated state that would re-notify everything.
	tmp, err := os.CreateTemp(dir, ".notify-*.json")
	if err != nil {
		return fmt.Errorf("criar arquivo temporário de estado: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("gravar estado de notificações: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("fechar estado de notificações: %w", err)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return fmt.Errorf("ajustar permissões do estado: %w", err)
	}
	if err := os.Rename(tmpName, f.path); err != nil {
		return fmt.Errorf("salvar estado de notificações: %w", err)
	}
	return nil
}
