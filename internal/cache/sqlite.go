// Package cache provides a local key-value store with TTL, backed by
// SQLite (modernc.org/sqlite, CGO-free), used to cache 42 API responses.
package cache

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	// Registers the CGO-free "sqlite" database/sql driver.
	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS cache (
	key        TEXT PRIMARY KEY,
	value      BLOB NOT NULL,
	expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_cache_expires_at ON cache (expires_at);
`

// Store is a SQLite-backed cache with per-entry TTL.
// It is safe for concurrent use.
type Store struct {
	db *sql.DB
}

// Open creates (if needed) and opens the cache database at path,
// creating parent directories and applying the schema.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("criar diretório de cache: %w", err)
	}

	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("abrir cache: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrar cache: %w", err)
	}

	return &Store{db: db}, nil
}

// Get returns the value for key. The second return is false on miss
// (absent or expired entry).
func (s *Store) Get(key string) ([]byte, bool, error) {
	var value []byte
	err := s.db.QueryRow(
		"SELECT value FROM cache WHERE key = ? AND expires_at > ?",
		key, time.Now().Unix(),
	).Scan(&value)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("ler cache %q: %w", key, err)
	}
	return value, true, nil
}

// Set stores value under key with the given TTL, replacing any previous entry.
func (s *Store) Set(key string, value []byte, ttl time.Duration) error {
	_, err := s.db.Exec(
		"INSERT INTO cache (key, value, expires_at) VALUES (?, ?, ?) "+
			"ON CONFLICT(key) DO UPDATE SET value = excluded.value, expires_at = excluded.expires_at",
		key, value, time.Now().Add(ttl).Unix(),
	)
	if err != nil {
		return fmt.Errorf("gravar cache %q: %w", key, err)
	}
	return nil
}

// Clear removes all entries.
func (s *Store) Clear() error {
	if _, err := s.db.Exec("DELETE FROM cache"); err != nil {
		return fmt.Errorf("limpar cache: %w", err)
	}
	return nil
}

// Close releases the underlying database handle.
func (s *Store) Close() error {
	return s.db.Close()
}
