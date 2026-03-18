package memory

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (no CGo)
)

// SQLiteStore persists conversation history in a local SQLite database.
//
// Each message row carries user_id, role, content, and timestamp. History is
// capped at maxHistory rows per user: on every write the oldest rows beyond
// the cap are pruned.
type SQLiteStore struct {
	db         *sql.DB
	maxHistory int
}

// NewSQLiteStore opens (or creates) the SQLite database at dbPath and
// auto-creates the schema. dbPath may contain a leading "~" which is expanded
// to the user's home directory.
func NewSQLiteStore(dbPath string, maxHistory int) (*SQLiteStore, error) {
	if maxHistory <= 0 {
		maxHistory = 50
	}

	// Expand leading ~
	if strings.HasPrefix(dbPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("sqlite: expand home dir: %w", err)
		}
		dbPath = filepath.Join(home, dbPath[2:])
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("sqlite: create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open %s: %w", dbPath, err)
	}

	if err := createSchema(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite: create schema: %w", err)
	}

	return &SQLiteStore{db: db, maxHistory: maxHistory}, nil
}

func createSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id   TEXT    NOT NULL,
			role      TEXT    NOT NULL,
			content   TEXT    NOT NULL,
			timestamp TEXT    NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_messages_user_id ON messages(user_id);
	`)
	return err
}

// AddMessage appends a message to the user's history and prunes rows beyond maxHistory.
func (s *SQLiteStore) AddMessage(ctx context.Context, userID string, msg Message) error {
	ts := msg.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO messages (user_id, role, content, timestamp) VALUES (?, ?, ?, ?)`,
		userID, msg.Role, msg.Content, ts.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("sqlite: insert message: %w", err)
	}

	// Prune oldest rows beyond cap.
	_, err = s.db.ExecContext(ctx, `
		DELETE FROM messages
		WHERE user_id = ?
		  AND id NOT IN (
		      SELECT id FROM messages
		      WHERE user_id = ?
		      ORDER BY id DESC
		      LIMIT ?
		  )`, userID, userID, s.maxHistory)
	if err != nil {
		return fmt.Errorf("sqlite: prune history: %w", err)
	}

	return nil
}

// GetHistory returns up to limit messages for the user in chronological order.
// If limit <= 0 the store's maxHistory is used.
func (s *SQLiteStore) GetHistory(ctx context.Context, userID string, limit int) ([]Message, error) {
	if limit <= 0 {
		limit = s.maxHistory
	}

	// Select newest-first, then reverse for chronological output.
	rows, err := s.db.QueryContext(ctx, `
		SELECT role, content, timestamp
		FROM messages
		WHERE user_id = ?
		ORDER BY id DESC
		LIMIT ?`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("sqlite: query history: %w", err)
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		var tsStr string
		if err := rows.Scan(&m.Role, &m.Content, &tsStr); err != nil {
			return nil, fmt.Errorf("sqlite: scan row: %w", err)
		}
		if t, err := time.Parse(time.RFC3339, tsStr); err == nil {
			m.Timestamp = t
		}
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: iterate rows: %w", err)
	}

	// Reverse to chronological order.
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

// ClearHistory deletes all message history for the given user.
func (s *SQLiteStore) ClearHistory(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM messages WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("sqlite: clear history: %w", err)
	}
	return nil
}

// Close closes the underlying database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
