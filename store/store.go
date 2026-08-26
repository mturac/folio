// Package store is Folio's local SQLite library. Chats, screenshots
// and newsletters live on the user's disk — no cloud, no account.
package store

import (
	"database/sql"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	KindChat   = "chat"
	KindShot   = "shot"
	KindLetter = "letter"
)

type Item struct {
	ID     int64
	Kind   string
	Source string
	Title  string
	Body   string
	When   time.Time
}

type DB struct{ sql *sql.DB }

func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// One writer at a time; busy_timeout waits instead of SQLITE_BUSY.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		db.Close()
		return nil, err
	}
	schema := `
CREATE TABLE IF NOT EXISTS items (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	kind TEXT NOT NULL,
	source TEXT NOT NULL UNIQUE,
	title TEXT NOT NULL DEFAULT '',
	body TEXT NOT NULL DEFAULT '',
	saved_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE VIRTUAL TABLE IF NOT EXISTS items_fts USING fts5(
	title, body, content='items', content_rowid='id'
);
CREATE TRIGGER IF NOT EXISTS trg_ai AFTER INSERT ON items BEGIN
	INSERT INTO items_fts(rowid, title, body) VALUES (new.id, new.title, new.body);
END;
`
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &DB{sql: db}, nil
}

// Close releases the SQLite handle. A second call is a no-op.
func (d *DB) Close() error {
	if d.sql == nil {
		return nil
	}
	err := d.sql.Close()
	d.sql = nil
	return err
}

func (d *DB) Add(it Item) (int64, error) {
	res, err := d.sql.Exec(
		`INSERT OR IGNORE INTO items (kind, source, title, body) VALUES (?, ?, ?, ?)`,
		it.Kind, it.Source, it.Title, it.Body,
	)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if n == 0 {
		var id int64
		if err := d.sql.QueryRow(`SELECT id FROM items WHERE source=?`, it.Source).Scan(&id); err != nil {
			return 0, err
		}
		return id, nil
	}
	return res.LastInsertId()
}

func (d *DB) Search(q string) ([]Item, error) {
	safe := sanitizeFTS5(q)
	rows, err := d.sql.Query(
		`SELECT items.id, items.kind, items.source, items.title, items.body
		 FROM items_fts JOIN items ON items.id = items_fts.rowid
		 WHERE items_fts MATCH ? ORDER BY rank LIMIT 50`,
		safe,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Item, 0, 16)
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.Kind, &it.Source, &it.Title, &it.Body); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (d *DB) Get(id int64) (*Item, error) {
	var it Item
	err := d.sql.QueryRow(
		`SELECT id, kind, source, title, body FROM items WHERE id=?`, id,
	).Scan(&it.ID, &it.Kind, &it.Source, &it.Title, &it.Body)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &it, nil
}

func (d *DB) Count() (int, error) {
	var n int
	err := d.sql.QueryRow(`SELECT COUNT(*) FROM items`).Scan(&n)
	return n, err
}

func (d *DB) List(kind string, limit int) ([]Item, error) {
	if limit <= 0 {
		limit = 50
	}
	q := `SELECT id, kind, source, title, body FROM items`
	args := []any{}
	if kind != "" {
		q += ` WHERE kind=?`
		args = append(args, kind)
	}
	q += ` ORDER BY saved_at DESC, id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := d.sql.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Item, 0, 16)
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.Kind, &it.Source, &it.Title, &it.Body); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// sanitizeFTS5 quotes each whitespace token so user input cannot be
// parsed as FTS5 operators (OR, NEAR, unmatched parens).
func sanitizeFTS5(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return `""`
	}
	fields := strings.Fields(q)
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.ReplaceAll(f, `"`, `""`)
		parts = append(parts, `"`+f+`"`)
	}
	return strings.Join(parts, " ")
}
