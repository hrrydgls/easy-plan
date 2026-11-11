// Package store holds all SQLite access for Easy Plan.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver" // database/sql driver "sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"  // bundled SQLite build, no cgo
)

// ErrNotFound is returned when a row doesn't exist or isn't owned by the
// requesting user. Callers should treat both cases the same (404).
var ErrNotFound = errors.New("not found")

// ErrEmailTaken is returned by CreateUser when the email is already registered.
var ErrEmailTaken = errors.New("email already registered")

type Store struct {
	db *sql.DB
}

type User struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
	CreatedAt    int64  `json:"createdAt"`
}

type Calendar struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"createdAt"`
}

// EventType is a kind of day a user can tag ("Work", "Holiday", ...).
// Count is "none", "month" or "year": how its days are totalled.
type EventType struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Emoji    string `json:"emoji"`
	Count    string `json:"count"`
	Position int    `json:"position"`
}

func Open(path string) (*Store, error) {
	dsn := "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(1) // sqlite: one writer, avoids lock contention

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id            INTEGER PRIMARY KEY,
			name          TEXT NOT NULL,
			email         TEXT NOT NULL UNIQUE COLLATE NOCASE,
			password_hash TEXT NOT NULL,
			created_at    INTEGER NOT NULL
		);

		CREATE TABLE IF NOT EXISTS sessions (
			token_hash TEXT PRIMARY KEY,
			user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			expires_at INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS sessions_user ON sessions(user_id);

		CREATE TABLE IF NOT EXISTS calendars (
			id         INTEGER PRIMARY KEY,
			user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name       TEXT NOT NULL,
			created_at INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS calendars_user ON calendars(user_id);

		CREATE TABLE IF NOT EXISTS event_types (
			id          INTEGER PRIMARY KEY,
			calendar_id INTEGER NOT NULL REFERENCES calendars(id) ON DELETE CASCADE,
			name        TEXT NOT NULL,
			emoji       TEXT NOT NULL,
			count_mode  TEXT NOT NULL DEFAULT 'none' CHECK (count_mode IN ('none','month','year')),
			position    INTEGER NOT NULL DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS event_types_calendar ON event_types(calendar_id);

		CREATE TABLE IF NOT EXISTS day_tags (
			calendar_id   INTEGER NOT NULL REFERENCES calendars(id) ON DELETE CASCADE,
			day           TEXT NOT NULL, -- YYYY-MM-DD
			event_type_id INTEGER NOT NULL REFERENCES event_types(id) ON DELETE CASCADE,
			PRIMARY KEY (calendar_id, day, event_type_id)
		);
	`)
	return err
}

// ---------- Users ----------

func (s *Store) CreateUser(ctx context.Context, name, email, hash string) (*User, error) {
	now := time.Now().Unix()
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO users (name, email, password_hash, created_at) VALUES (?, ?, ?, ?)`,
		name, email, hash, now)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &User{ID: id, Name: name, Email: email, PasswordHash: hash, CreatedAt: now}, nil
}

func (s *Store) UserByEmail(ctx context.Context, email string) (*User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx,
		`SELECT id, name, email, password_hash, created_at FROM users WHERE email = ?`, email))
}

func (s *Store) UserByID(ctx context.Context, id int64) (*User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx,
		`SELECT id, name, email, password_hash, created_at FROM users WHERE id = ?`, id))
}

func (s *Store) scanUser(row *sql.Row) (*User, error) {
	var u User
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

// ---------- Sessions ----------

func (s *Store) CreateSession(ctx context.Context, tokenHash string, userID int64, expires time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES (?, ?, ?)`,
		tokenHash, userID, expires.Unix())
	return err
}

// SessionUser returns the user for a live session, or ErrNotFound.
func (s *Store) SessionUser(ctx context.Context, tokenHash string) (*User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx, `
		SELECT u.id, u.name, u.email, u.password_hash, u.created_at
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ? AND s.expires_at > ?`, tokenHash, time.Now().Unix()))
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash)
	return err
}

func (s *Store) PruneSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= ?`, time.Now().Unix())
	return err
}

// ---------- Calendars ----------

func (s *Store) ListCalendars(ctx context.Context, userID int64) ([]Calendar, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, created_at FROM calendars WHERE user_id = ? ORDER BY created_at, id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Calendar{}
	for rows.Next() {
		var c Calendar
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) GetCalendar(ctx context.Context, userID, calID int64) (*Calendar, error) {
	var c Calendar
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, created_at FROM calendars WHERE id = ? AND user_id = ?`, calID, userID).
		Scan(&c.ID, &c.Name, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &c, err
}

// CreateCalendar makes a fresh calendar, optionally seeded with starter event types.
func (s *Store) CreateCalendar(ctx context.Context, userID int64, name string, starter []EventType) (*Calendar, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	now := time.Now().Unix()
	res, err := tx.ExecContext(ctx, `INSERT INTO calendars (user_id, name, created_at) VALUES (?, ?, ?)`, userID, name, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	for i, t := range starter {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO event_types (calendar_id, name, emoji, count_mode, position) VALUES (?, ?, ?, ?, ?)`,
			id, t.Name, t.Emoji, t.Count, i); err != nil {
			return nil, err
		}
	}
	return &Calendar{ID: id, Name: name, CreatedAt: now}, tx.Commit()
}

func (s *Store) RenameCalendar(ctx context.Context, userID, calID int64, name string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE calendars SET name = ? WHERE id = ? AND user_id = ?`, name, calID, userID)
	return affected(res, err)
}

func (s *Store) DeleteCalendar(ctx context.Context, userID, calID int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM calendars WHERE id = ? AND user_id = ?`, calID, userID)
	return affected(res, err)
}

func affected(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ---------- Event types ----------

func (s *Store) ListEventTypes(ctx context.Context, calID int64) ([]EventType, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, emoji, count_mode, position FROM event_types WHERE calendar_id = ? ORDER BY position, id`, calID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []EventType{}
	for rows.Next() {
		var t EventType
		if err := rows.Scan(&t.ID, &t.Name, &t.Emoji, &t.Count, &t.Position); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ReplaceEventTypes makes the calendar's event types exactly `types`, in order.
// Types with an ID that belongs to the calendar are updated; types with ID 0
// (or an unknown ID) are created; existing types not in the list are deleted,
// which also removes them from every tagged day.
func (s *Store) ReplaceEventTypes(ctx context.Context, calID int64, types []EventType) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	existing := map[int64]bool{}
	rows, err := tx.QueryContext(ctx, `SELECT id FROM event_types WHERE calendar_id = ?`, calID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		existing[id] = true
	}
	rows.Close()

	keep := map[int64]bool{}
	for i, t := range types {
		if t.ID != 0 && existing[t.ID] {
			if _, err := tx.ExecContext(ctx,
				`UPDATE event_types SET name = ?, emoji = ?, count_mode = ?, position = ? WHERE id = ? AND calendar_id = ?`,
				t.Name, t.Emoji, t.Count, i, t.ID, calID); err != nil {
				return err
			}
			keep[t.ID] = true
			continue
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO event_types (calendar_id, name, emoji, count_mode, position) VALUES (?, ?, ?, ?, ?)`,
			calID, t.Name, t.Emoji, t.Count, i); err != nil {
			return err
		}
	}
	for id := range existing {
		if !keep[id] {
			if _, err := tx.ExecContext(ctx, `DELETE FROM event_types WHERE id = ? AND calendar_id = ?`, id, calID); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// ---------- Day tags ----------

// Days returns every tagged day of a calendar: "YYYY-MM-DD" -> event type IDs.
func (s *Store) Days(ctx context.Context, calID int64) (map[string][]int64, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT d.day, d.event_type_id FROM day_tags d
		JOIN event_types t ON t.id = d.event_type_id
		WHERE d.calendar_id = ? ORDER BY d.day, t.position, t.id`, calID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]int64{}
	for rows.Next() {
		var day string
		var id int64
		if err := rows.Scan(&day, &id); err != nil {
			return nil, err
		}
		out[day] = append(out[day], id)
	}
	return out, rows.Err()
}

// SetDay replaces the event types tagged on one day. IDs that don't belong to
// the calendar are ignored. Returns the IDs actually stored.
func (s *Store) SetDay(ctx context.Context, calID int64, day string, typeIDs []int64) ([]int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM day_tags WHERE calendar_id = ? AND day = ?`, calID, day); err != nil {
		return nil, err
	}
	stored := []int64{}
	seen := map[int64]bool{}
	for _, id := range typeIDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		res, err := tx.ExecContext(ctx, `
			INSERT INTO day_tags (calendar_id, day, event_type_id)
			SELECT ?, ?, id FROM event_types WHERE id = ? AND calendar_id = ?`, calID, day, id, calID)
		if err != nil {
			return nil, err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			stored = append(stored, id)
		}
	}
	return stored, tx.Commit()
}
