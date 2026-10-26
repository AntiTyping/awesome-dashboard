package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type Item struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Subtitle  string    `json:"subtitle"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Store struct {
	db *sql.DB
}

const schema = `CREATE TABLE IF NOT EXISTS items (
	id         TEXT PRIMARY KEY,
	title      TEXT NOT NULL,
	subtitle   TEXT NOT NULL DEFAULT '',
	color      TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
)`

const cols = `id, title, subtitle, color, created_at, updated_at`

func Open(dbPath string) (*Store, error) {
	if dbPath != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
			return nil, fmt.Errorf("create db directory: %w", err)
		}
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	// A single connection keeps ":memory:" databases consistent and serializes writes.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}
	return &Store{db: db}, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanItem(r scanner) (Item, error) {
	var it Item
	var created, updated string
	if err := r.Scan(&it.ID, &it.Title, &it.Subtitle, &it.Color, &created, &updated); err != nil {
		return it, err
	}
	var err error
	if it.CreatedAt, err = time.Parse("2006-01-02 15:04:05", created); err != nil {
		return it, err
	}
	if it.UpdatedAt, err = time.Parse("2006-01-02 15:04:05", updated); err != nil {
		return it, err
	}
	return it, nil
}

func (s *Store) getByID(id string) (*Item, error) {
	it, err := scanItem(s.db.QueryRow(`SELECT `+cols+` FROM items WHERE id = ?`, id))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("item not found")
	}
	if err != nil {
		return nil, err
	}
	return &it, nil
}

func (s *Store) Add(title, subtitle, color string) (*Item, error) {
	id := uuid.New().String()
	_, err := s.db.Exec(
		`INSERT INTO items (id, title, subtitle, color, created_at, updated_at)
		 VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))`,
		id, title, subtitle, color)
	if err != nil {
		return nil, err
	}
	return s.getByID(id)
}

func (s *Store) GetAll() ([]Item, error) {
	rows, err := s.db.Query(`SELECT ` + cols + ` FROM items ORDER BY created_at, rowid`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Item{}
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (s *Store) GetByPrefix(prefix string) (*Item, error) {
	rows, err := s.db.Query(
		`SELECT `+cols+` FROM items WHERE substr(id, 1, length(?1)) = ?1`, prefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var matches []Item
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		matches = append(matches, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("item not found")
	case 1:
		return &matches[0], nil
	default:
		return nil, fmt.Errorf("ambiguous prefix, matches %d items", len(matches))
	}
}

func (s *Store) Update(id string, title, subtitle, color *string) (*Item, error) {
	sets := []string{"updated_at = datetime('now')"}
	args := []any{}
	if title != nil {
		sets = append(sets, "title = ?")
		args = append(args, *title)
	}
	if subtitle != nil {
		sets = append(sets, "subtitle = ?")
		args = append(args, *subtitle)
	}
	if color != nil {
		sets = append(sets, "color = ?")
		args = append(args, *color)
	}
	args = append(args, id)
	res, err := s.db.Exec(`UPDATE items SET `+strings.Join(sets, ", ")+` WHERE id = ?`, args...)
	if err != nil {
		return nil, err
	}
	if n, err := res.RowsAffected(); err != nil {
		return nil, err
	} else if n == 0 {
		return nil, fmt.Errorf("item not found")
	}
	return s.getByID(id)
}

func (s *Store) Delete(id string) error {
	res, err := s.db.Exec(`DELETE FROM items WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return fmt.Errorf("item not found")
	}
	return nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
