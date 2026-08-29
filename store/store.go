// Package store is Folio's local SQLite library. Chats, screenshots
// and newsletters live on the user's disk — no cloud, no account.
package store

import (
	"context"
	"database/sql"
	"fmt"
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
	ID     int64     `json:"ID"`
	Kind   string    `json:"Kind"`
	Source string    `json:"Source"`
	Title  string    `json:"Title"`
	Body   string    `json:"Body"`
	When   time.Time `json:"When"`
}

type Stats struct {
	Total  int            `json:"total"`
	ByKind map[string]int `json:"byKind"`
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
	saved_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	occurred_at DATETIME
);
CREATE VIRTUAL TABLE IF NOT EXISTS items_fts USING fts5(
	title, body, content='items', content_rowid='id'
);
CREATE TRIGGER IF NOT EXISTS trg_ai AFTER INSERT ON items BEGIN
	INSERT INTO items_fts(rowid, title, body) VALUES (new.id, new.title, new.body);
END;
CREATE TRIGGER IF NOT EXISTS trg_ad AFTER DELETE ON items BEGIN
	INSERT INTO items_fts(items_fts, rowid, title, body) VALUES('delete', old.id, old.title, old.body);
END;
CREATE TRIGGER IF NOT EXISTS trg_au AFTER UPDATE ON items BEGIN
	INSERT INTO items_fts(items_fts, rowid, title, body) VALUES('delete', old.id, old.title, old.body);
	INSERT INTO items_fts(rowid, title, body) VALUES (new.id, new.title, new.body);
END;
`
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return &DB{sql: db}, nil
}

func migrate(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(items)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	cols := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		cols[name] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !cols["occurred_at"] {
		if _, err := db.Exec(`ALTER TABLE items ADD COLUMN occurred_at DATETIME`); err != nil {
			return fmt.Errorf("migrate occurred_at: %w", err)
		}
	}
	return nil
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
		`INSERT OR IGNORE INTO items (kind, source, title, body, occurred_at) VALUES (?, ?, ?, ?, ?)`,
		it.Kind, it.Source, it.Title, it.Body, nullTime(it.When),
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

// Upsert inserts or replaces by source so re-ingest refreshes body/OCR/title.
func (d *DB) Upsert(it Item) (int64, error) {
	res, err := d.sql.Exec(
		`UPDATE items SET kind=?, title=?, body=?, occurred_at=?, saved_at=CURRENT_TIMESTAMP WHERE source=?`,
		it.Kind, it.Title, it.Body, nullTime(it.When), it.Source,
	)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if n > 0 {
		var id int64
		if err := d.sql.QueryRow(`SELECT id FROM items WHERE source=?`, it.Source).Scan(&id); err != nil {
			return 0, err
		}
		return id, nil
	}
	return d.Add(it)
}

func (d *DB) Delete(id int64) (bool, error) {
	res, err := d.sql.Exec(`DELETE FROM items WHERE id=?`, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (d *DB) DeleteBySource(source string) (bool, error) {
	res, err := d.sql.Exec(`DELETE FROM items WHERE source=?`, source)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// MsgSource builds a per-message source id under a chat thread export path.
func MsgSource(thread string, seq int) string {
	return fmt.Sprintf("%s#m%06d", thread, seq)
}

// IsMsgSource reports whether source is a per-message chat row.
func IsMsgSource(source string) bool {
	_, _, ok := SplitMsgSource(source)
	return ok
}

// SplitMsgSource returns thread path and 1-based seq for path#m000001.
func SplitMsgSource(source string) (thread string, seq int, ok bool) {
	i := strings.LastIndex(source, "#m")
	if i < 0 || i+2 >= len(source) {
		return "", 0, false
	}
	num := source[i+2:]
	if len(num) == 0 {
		return "", 0, false
	}
	for _, r := range num {
		if r < '0' || r > '9' {
			return "", 0, false
		}
	}
	var n int
	fmt.Sscanf(num, "%d", &n)
	if n < 1 {
		return "", 0, false
	}
	return source[:i], n, true
}

// ReplaceChat deletes a thread and its #m* messages, then inserts the
// thread summary plus each message. Returns the number of message rows.
func (d *DB) ReplaceChat(thread Item, msgs []Item) (int, error) {
	tx, err := d.sql.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM items WHERE source=? OR source LIKE ?`, thread.Source, thread.Source+"#m%"); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(
		`INSERT INTO items (kind, source, title, body, occurred_at) VALUES (?, ?, ?, ?, ?)`,
		thread.Kind, thread.Source, thread.Title, thread.Body, nullTime(thread.When),
	); err != nil {
		return 0, err
	}
	for _, m := range msgs {
		if _, err := tx.Exec(
			`INSERT INTO items (kind, source, title, body, occurred_at) VALUES (?, ?, ?, ?, ?)`,
			m.Kind, m.Source, m.Title, m.Body, nullTime(m.When),
		); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(msgs), nil
}

func (d *DB) Search(q string) ([]Item, error) {
	return d.SearchContext(context.Background(), q, "")
}

func (d *DB) SearchContext(ctx context.Context, q string, kind string) ([]Item, error) {
	safe := sanitizeFTS5(q)
	query := `SELECT items.id, items.kind, items.source, items.title, items.body, items.occurred_at
		 FROM items_fts JOIN items ON items.id = items_fts.rowid
		 WHERE items_fts MATCH ?`
	args := []any{safe}
	if kind != "" {
		query += ` AND items.kind=?`
		args = append(args, kind)
	}
	query += ` ORDER BY rank LIMIT 50`
	rows, err := d.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanItems(rows)
}

func (d *DB) Get(id int64) (*Item, error) {
	return d.GetContext(context.Background(), id)
}

func (d *DB) GetContext(ctx context.Context, id int64) (*Item, error) {
	var it Item
	var when sql.NullString
	err := d.sql.QueryRowContext(ctx,
		`SELECT id, kind, source, title, body, occurred_at FROM items WHERE id=?`, id,
	).Scan(&it.ID, &it.Kind, &it.Source, &it.Title, &it.Body, &when)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	it.When = parseTime(when)
	return &it, nil
}

func (d *DB) Count() (int, error) {
	var n int
	err := d.sql.QueryRow(`SELECT COUNT(*) FROM items`).Scan(&n)
	return n, err
}

func (d *DB) Stats() (Stats, error) {
	s := Stats{ByKind: map[string]int{}}
	// Chat stats count threads only; per-message rows stay searchable.
	rows, err := d.sql.Query(`
		SELECT kind, COUNT(*) FROM items
		WHERE source NOT LIKE '%#m%'
		GROUP BY kind`)
	if err != nil {
		return s, err
	}
	defer rows.Close()
	for rows.Next() {
		var kind string
		var n int
		if err := rows.Scan(&kind, &n); err != nil {
			return s, err
		}
		s.ByKind[kind] = n
		s.Total += n
	}
	return s, rows.Err()
}

func (d *DB) List(kind string, limit int) ([]Item, error) {
	return d.ListContext(context.Background(), kind, limit)
}

func (d *DB) ListContext(ctx context.Context, kind string, limit int) ([]Item, error) {
	if limit <= 0 {
		limit = 50
	}
	q := `SELECT id, kind, source, title, body, occurred_at FROM items WHERE source NOT LIKE '%#m%'`
	args := []any{}
	if kind != "" {
		q += ` AND kind=?`
		args = append(args, kind)
	}
	q += ` ORDER BY COALESCE(occurred_at, saved_at) DESC, id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := d.sql.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanItems(rows)
}

func scanItems(rows *sql.Rows) ([]Item, error) {
	out := make([]Item, 0, 16)
	for rows.Next() {
		var it Item
		var when sql.NullString
		if err := rows.Scan(&it.ID, &it.Kind, &it.Source, &it.Title, &it.Body, &when); err != nil {
			return nil, err
		}
		it.When = parseTime(when)
		out = append(out, it)
	}
	return out, rows.Err()
}

func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}

func parseTime(ns sql.NullString) time.Time {
	if !ns.Valid || ns.String == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", time.RFC3339Nano} {
		if t, err := time.Parse(layout, ns.String); err == nil {
			return t
		}
	}
	return time.Time{}
}

// sanitizeFTS5 quotes tokens after stripping FTS5 metacharacters
// (", *, :, (, ), ^) so column filters, prefix *, NEAR, and unmatched
// parens cannot be injected. Remaining text is AND-matched as literals.
func sanitizeFTS5(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return `""`
	}
	fields := strings.Fields(q)
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		cleaned := strings.Map(func(r rune) rune {
			switch r {
			case '"', '*', ':', '(', ')', '^':
				return -1
			}
			return r
		}, f)
		if cleaned == "" {
			continue
		}
		parts = append(parts, `"`+cleaned+`"`)
	}
	if len(parts) == 0 {
		return `""`
	}
	return strings.Join(parts, " ")
}
