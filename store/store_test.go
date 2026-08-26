package store

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAddAndSearchFindsBody(t *testing.T) {
	d, err := Open(filepath.Join(t.TempDir(), "folio.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { d.Close() })

	id, err := d.Add(Item{
		Kind:   KindChat,
		Source: "export/_chat.txt",
		Title:  "Family group",
		Body:   "Can you send the boarding pass for the Friday flight?",
	})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}

	hits, err := d.Search("boarding pass")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("want 1 hit, got %d", len(hits))
	}
	if hits[0].Kind != KindChat || !strings.Contains(hits[0].Body, "boarding") {
		t.Fatalf("wrong hit: %+v", hits[0])
	}
}

func TestDuplicateSourceIsIgnored(t *testing.T) {
	d, err := Open(filepath.Join(t.TempDir(), "folio.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { d.Close() })

	a := Item{Kind: KindShot, Source: "/shots/wifi.png", Title: "wifi", Body: "password"}
	id1, err := d.Add(a)
	if err != nil {
		t.Fatalf("add1: %v", err)
	}
	id2, err := d.Add(a)
	if err != nil {
		t.Fatalf("add2: %v", err)
	}
	if id1 != id2 {
		t.Fatalf("dedup should return same id: %d vs %d", id1, id2)
	}
	n, err := d.Count()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("want 1 row, got %d", n)
	}
}
