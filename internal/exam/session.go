package exam

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Mode distinguishes a timed exam from free practice of a single exercise.
type Mode string

// Session modes.
const (
	ModeExam     Mode = "exam"
	ModePractice Mode = "practice"
)

// Session is the state of the exam (or practice) in progress. It is persisted
// between runs so closing the terminal does not lose the exam.
type Session struct {
	Mode     Mode   `json:"mode"`
	Rank     string `json:"rank"`
	Level    int    `json:"level"`
	Exercise string `json:"exercise"`
	// Score goes from 0 to 100; each level passed adds an equal share.
	Score int `json:"score"`
	// Attempts counts the gradings of the current exercise.
	Attempts  int       `json:"attempts"`
	StartedAt time.Time `json:"started_at"`
	// Deadline is zero in practice mode (no timer).
	Deadline time.Time `json:"deadline,omitzero"`
	// Workspace holds subjects/, rendu/ and traces/.
	Workspace string `json:"workspace"`
}

// Remaining returns the time left before the deadline (never negative).
// It is zero when there is no deadline.
func (s Session) Remaining(now time.Time) time.Duration {
	if s.Deadline.IsZero() || !now.Before(s.Deadline) {
		return 0
	}
	return s.Deadline.Sub(now)
}

// TimeUp reports whether a timed session ran past its deadline.
func (s Session) TimeUp(now time.Time) bool {
	return !s.Deadline.IsZero() && !now.Before(s.Deadline)
}

// SessionFile persists the session as JSON. A missing file means no session.
type SessionFile struct {
	path string
}

// NewSessionFile returns a store backed by the given file path.
func NewSessionFile(path string) *SessionFile {
	return &SessionFile{path: path}
}

// Load returns the saved session, or nil when there is none.
func (f *SessionFile) Load() (*Session, error) {
	data, err := os.ReadFile(f.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ler sessão de prova: %w", err)
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("sessão de prova corrompida (%s): %w", f.path, err)
	}
	return &s, nil
}

// Save writes the session atomically (temp file + rename).
func (f *SessionFile) Save(s Session) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("serializar sessão de prova: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(f.path), 0o700); err != nil {
		return fmt.Errorf("criar diretório da sessão: %w", err)
	}
	tmp := f.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("gravar sessão de prova: %w", err)
	}
	if err := os.Rename(tmp, f.path); err != nil {
		return fmt.Errorf("salvar sessão de prova: %w", err)
	}
	return nil
}

// Clear removes the saved session; clearing when there is none is fine.
func (f *SessionFile) Clear() error {
	if err := os.Remove(f.path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("apagar sessão de prova: %w", err)
	}
	return nil
}
