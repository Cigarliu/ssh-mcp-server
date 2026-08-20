// Package state owns the local, multi-process persistent state for sshmcp.
package state

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const maxStoredOutputBytes = 16 * 1024

// ConnectionProfile contains server-side connection data. Callers must not
// return this structure directly to MCP clients because it contains targets
// and authentication material.
type ConnectionProfile struct {
	ID             string
	Description    string
	Host           string
	Port           int
	Username       string
	Password       string
	PrivateKeyPath string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// HistoryEntry is a durable record of a command or terminal interaction.
type HistoryEntry struct {
	ID                  int64
	ConnectionID        string
	DescriptionSnapshot string
	Kind                string
	Input               string
	Output              string
	State               string
	ExitCode            *int
	CreatedAt           time.Time
}

// Store is safe for concurrent use by multiple goroutines and SQLite-backed
// processes. WAL allows readers to proceed while another process writes.
type Store struct {
	db      *sql.DB
	writeMu sync.Mutex
}

// Open creates the database and applies the small forward-only schema.
func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("state database path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("create state directory: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open state database: %w", err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	store := &Store{db: db}
	if err := store.initialize(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) initialize() error {
	for _, statement := range []string{
		"PRAGMA busy_timeout = 5000",
		"PRAGMA journal_mode = WAL",
		"PRAGMA foreign_keys = ON",
		`CREATE TABLE IF NOT EXISTS connection_profiles (
			connection_id TEXT PRIMARY KEY,
			description TEXT NOT NULL,
			host TEXT NOT NULL,
			port INTEGER NOT NULL,
			username TEXT NOT NULL,
			password TEXT NOT NULL DEFAULT '',
			private_key_path TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS execution_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			connection_id TEXT NOT NULL,
			description_snapshot TEXT NOT NULL,
			kind TEXT NOT NULL,
			input TEXT NOT NULL,
			output TEXT NOT NULL,
			state TEXT NOT NULL,
			exit_code INTEGER,
			created_at INTEGER NOT NULL,
			FOREIGN KEY (connection_id) REFERENCES connection_profiles(connection_id)
		)`,
		"CREATE INDEX IF NOT EXISTS execution_history_connection_created_idx ON execution_history(connection_id, created_at DESC, id DESC)",
	} {
		if _, err := s.db.Exec(statement); err != nil {
			return fmt.Errorf("initialize state database: %w", err)
		}
	}
	return nil
}

func (s *Store) Close() error { return s.db.Close() }

// SeedProfiles imports legacy YAML profiles without overwriting profiles that
// were already edited through a newer server instance.
func (s *Store) SeedProfiles(profiles []ConnectionProfile) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin profile seed: %w", err)
	}
	defer tx.Rollback()
	now := time.Now().UTC().UnixMilli()
	for _, profile := range profiles {
		if strings.TrimSpace(profile.Description) == "" {
			profile.Description = "Saved SSH connection " + profile.ID
		}
		if profile.Port == 0 {
			profile.Port = 22
		}
		_, err = tx.Exec(`INSERT INTO connection_profiles
			(connection_id, description, host, port, username, password, private_key_path, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(connection_id) DO NOTHING`,
			profile.ID, profile.Description, profile.Host, profile.Port, profile.Username,
			profile.Password, profile.PrivateKeyPath, now, now)
		if err != nil {
			return fmt.Errorf("seed profile %q: %w", profile.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit profile seed: %w", err)
	}
	return nil
}

func (s *Store) GetProfile(id string) (ConnectionProfile, error) {
	row := s.db.QueryRow(`SELECT connection_id, description, host, port, username, password, private_key_path, created_at, updated_at
		FROM connection_profiles WHERE connection_id = ?`, id)
	return scanProfile(row)
}

func (s *Store) ListProfiles() ([]ConnectionProfile, error) {
	rows, err := s.db.Query(`SELECT connection_id, description, host, port, username, password, private_key_path, created_at, updated_at
		FROM connection_profiles ORDER BY connection_id`)
	if err != nil {
		return nil, fmt.Errorf("list connection profiles: %w", err)
	}
	defer rows.Close()
	profiles := make([]ConnectionProfile, 0)
	for rows.Next() {
		profile, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate connection profiles: %w", err)
	}
	return profiles, nil
}

func (s *Store) CreateProfile(profile ConnectionProfile) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	now := time.Now().UTC()
	if profile.Port == 0 {
		profile.Port = 22
	}
	_, err := s.db.Exec(`INSERT INTO connection_profiles
		(connection_id, description, host, port, username, password, private_key_path, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		profile.ID, profile.Description, profile.Host, profile.Port, profile.Username,
		profile.Password, profile.PrivateKeyPath, now.UnixMilli(), now.UnixMilli())
	if err != nil {
		return fmt.Errorf("create connection profile %q: %w", profile.ID, err)
	}
	return nil
}

func (s *Store) DeleteProfile(id string) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	result, err := s.db.Exec("DELETE FROM connection_profiles WHERE connection_id = ?", id)
	if err != nil {
		return fmt.Errorf("delete connection profile %q: %w", id, err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return fmt.Errorf("connection profile %q not found", id)
	}
	return nil
}

func (s *Store) RecordHistory(entry HistoryEntry) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if len(entry.Output) > maxStoredOutputBytes {
		entry.Output = entry.Output[:maxStoredOutputBytes] + "\n[output truncated by sshmcp]"
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.Exec(`INSERT INTO execution_history
		(connection_id, description_snapshot, kind, input, output, state, exit_code, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ConnectionID, entry.DescriptionSnapshot, entry.Kind, entry.Input,
		entry.Output, entry.State, entry.ExitCode, entry.CreatedAt.UnixMilli())
	if err != nil {
		return fmt.Errorf("record execution history: %w", err)
	}
	return nil
}

func (s *Store) ListHistory(connectionID string, limit int) ([]HistoryEntry, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.Query(`SELECT id, connection_id, description_snapshot, kind, input, output, state, exit_code, created_at
		FROM execution_history WHERE connection_id = ? ORDER BY created_at DESC, id DESC LIMIT ?`, connectionID, limit)
	if err != nil {
		return nil, fmt.Errorf("list execution history: %w", err)
	}
	defer rows.Close()
	entries := make([]HistoryEntry, 0)
	for rows.Next() {
		var entry HistoryEntry
		var createdAt int64
		if err := rows.Scan(&entry.ID, &entry.ConnectionID, &entry.DescriptionSnapshot, &entry.Kind,
			&entry.Input, &entry.Output, &entry.State, &entry.ExitCode, &createdAt); err != nil {
			return nil, fmt.Errorf("scan execution history: %w", err)
		}
		entry.CreatedAt = time.UnixMilli(createdAt).UTC()
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate execution history: %w", err)
	}
	return entries, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanProfile(row rowScanner) (ConnectionProfile, error) {
	var profile ConnectionProfile
	var createdAt, updatedAt int64
	if err := row.Scan(&profile.ID, &profile.Description, &profile.Host, &profile.Port, &profile.Username,
		&profile.Password, &profile.PrivateKeyPath, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return ConnectionProfile{}, fmt.Errorf("connection profile not found")
		}
		return ConnectionProfile{}, fmt.Errorf("scan connection profile: %w", err)
	}
	profile.CreatedAt = time.UnixMilli(createdAt).UTC()
	profile.UpdatedAt = time.UnixMilli(updatedAt).UTC()
	return profile, nil
}
