// Package store is Folio's local SQLite library. Chats, screenshots
// and newsletters live on the user's disk — no cloud, no account.
package store

import (
	"database/sql"
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
CREATE TRIGGER IF NOT EXISTS trg_au AFTER UPDATE ON items BEGIN
	INSERT INTO items_fts(items_fts, rowid, title, body)
	VALUES ('delete', old.id, old.title, old.body);
	INSERT INTO items_fts(rowid, title, body) VALUES (new.id, new.title, new.body);
END;
CREATE TRIGGER IF NOT EXISTS trg_ad AFTER DELETE ON items BEGIN
	INSERT INTO items_fts(items_fts, rowid, title, body)
	VALUES ('delete', old.id, old.title, old.body);
END;
`
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &DB{sql: db}, nil
}

func (d *DB) Close() error { return d.sql.Close() }

func (d *DB) Add(it Item) (int64, error) {
	res, err := d.sql.Exec(
		`INSERT OR IGNORE INTO items (kind, source, title, body) VALUES (?, ?, ?, ?)`,
		it.Kind, it.Source, it.Title, it.Body,
	)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		var id int64
		err := d.sql.QueryRow(`SELECT id FROM items WHERE source=?`, it.Source).Scan(&id)
		return id, err
	}
	return res.LastInsertId()
}

func (d *DB) Search(q string) ([]Item, error) {
	rows, err := d.sql.Query(
		`SELECT items.id, items.kind, items.source, items.title, items.body
		 FROM items_fts JOIN items ON items.id = items_fts.rowid
		 WHERE items_fts MATCH ? ORDER BY rank LIMIT 50`,
		q,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Item
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.Kind, &it.Source, &it.Title, &it.Body); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
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
	q += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := d.sql.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Item
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.Kind, &it.Source, &it.Title, &it.Body); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}
