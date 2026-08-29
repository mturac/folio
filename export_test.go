package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mturac/folio/store"
)

func TestExportJSONAndMarkdown(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "folio.db")
	d, err := store.Open(db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.Upsert(store.Item{Kind: store.KindLetter, Source: "a.html", Title: "Weekly", Body: "boarding tip"})
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	oldHome := os.Getenv("HOME")
	t.Cleanup(func() { os.Setenv("HOME", oldHome) })
	os.Setenv("HOME", dir)
	os.MkdirAll(filepath.Join(dir, ".folio"), 0o700)
	os.Rename(db, filepath.Join(dir, ".folio", "folio.db"))

	// JSON
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w
	if err := cmdExport([]string{"json"}); err != nil {
		t.Fatal(err)
	}
	w.Close()
	os.Stdout = old
	var items []store.Item
	if err := json.NewDecoder(r).Decode(&items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("json items=%d", len(items))
	}

	r, w, _ = os.Pipe()
	os.Stdout = w
	if err := cmdExport([]string{"md"}); err != nil {
		t.Fatal(err)
	}
	w.Close()
	os.Stdout = old
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	md := string(buf[:n])
	if !strings.Contains(md, "## [letter] Weekly") || !strings.Contains(md, "boarding tip") {
		t.Fatalf("md=%q", md)
	}
}

func TestDoctorRuns(t *testing.T) {
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	t.Cleanup(func() { os.Setenv("HOME", oldHome) })
	os.Setenv("HOME", dir)
	if err := cmdDoctor(); err != nil {
		t.Fatal(err)
	}
}
