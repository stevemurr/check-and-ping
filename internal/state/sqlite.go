package state

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLite implements persistent state tracking using SQLite
type SQLite struct {
	db *sql.DB
	mu sync.Mutex
}

// NewSQLite creates a new SQLite state tracker
func NewSQLite(dbPath string) (*SQLite, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Create table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS alert_state (
			check_name TEXT PRIMARY KEY,
			result_hash TEXT NOT NULL,
			alerted_at DATETIME NOT NULL
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create table: %w", err)
	}

	return &SQLite{db: db}, nil
}

// ShouldAlert returns true if this hash hasn't been alerted
func (s *SQLite) ShouldAlert(checkName string, resultHash string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	var storedHash string
	err := s.db.QueryRow(
		"SELECT result_hash FROM alert_state WHERE check_name = ?",
		checkName,
	).Scan(&storedHash)

	if err == sql.ErrNoRows {
		return true
	}
	if err != nil {
		// On error, err on the side of alerting
		return true
	}

	return storedHash != resultHash
}

// MarkAlerted records that an alert was sent
func (s *SQLite) MarkAlerted(checkName string, resultHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		INSERT INTO alert_state (check_name, result_hash, alerted_at)
		VALUES (?, ?, ?)
		ON CONFLICT(check_name) DO UPDATE SET
			result_hash = excluded.result_hash,
			alerted_at = excluded.alerted_at
	`, checkName, resultHash, time.Now())

	if err != nil {
		return fmt.Errorf("upsert alert state: %w", err)
	}

	return nil
}

// Clear removes state for a check
func (s *SQLite) Clear(checkName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("DELETE FROM alert_state WHERE check_name = ?", checkName)
	if err != nil {
		return fmt.Errorf("delete alert state: %w", err)
	}

	return nil
}

// Close closes the database connection
func (s *SQLite) Close() error {
	return s.db.Close()
}
