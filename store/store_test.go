package store

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
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
		When:   time.Date(2023, 12, 31, 10, 15, 0, 0, time.UTC),
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
	if hits[0].When.IsZero() {
		t.Fatal("when must be persisted")
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

func TestUpsertRefreshesBodyAndFTS(t *testing.T) {
	d, err := Open(filepath.Join(t.TempDir(), "folio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })

	id1, err := d.Upsert(Item{Kind: KindShot, Source: "/a.png", Title: "a", Body: "oldtoken"})
	if err != nil {
		t.Fatal(err)
	}
	id2, err := d.Upsert(Item{Kind: KindShot, Source: "/a.png", Title: "a", Body: "newtoken hunter2"})
	if err != nil {
		t.Fatal(err)
	}
	if id1 != id2 {
		t.Fatalf("upsert must keep id: %d vs %d", id1, id2)
	}
	old, err := d.Search("oldtoken")
	if err != nil {
		t.Fatal(err)
	}
	if len(old) != 0 {
		t.Fatalf("old FTS body must be gone, got %d", len(old))
	}
	hits, err := d.Search("hunter2")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("want refreshed hit, got %d", len(hits))
	}
}

func TestDeleteRemovesFromFTS(t *testing.T) {
	d, err := Open(filepath.Join(t.TempDir(), "folio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })

	id, err := d.Add(Item{Kind: KindLetter, Source: "x.html", Title: "x", Body: "secretphrase"})
	if err != nil {
		t.Fatal(err)
	}
	ok, err := d.Delete(id)
	if err != nil || !ok {
		t.Fatalf("delete: %v ok=%v", err, ok)
	}
	hits, err := d.Search("secretphrase")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 {
		t.Fatalf("deleted item must leave FTS, got %d", len(hits))
	}
}

func TestStatsByKind(t *testing.T) {
	d, err := Open(filepath.Join(t.TempDir(), "folio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	d.Add(Item{Kind: KindChat, Source: "c1", Title: "c", Body: "hello"})
	d.Add(Item{Kind: KindShot, Source: "s1", Title: "s", Body: "pic"})
	d.Add(Item{Kind: KindShot, Source: "s2", Title: "s2", Body: "pic2"})
	s, err := d.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if s.Total != 3 || s.ByKind[KindChat] != 1 || s.ByKind[KindShot] != 2 {
		t.Fatalf("stats=%+v", s)
	}
}

func TestSearchKindFilter(t *testing.T) {
	d, err := Open(filepath.Join(t.TempDir(), "folio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	d.Add(Item{Kind: KindChat, Source: "c", Title: "c", Body: "boarding chat"})
	d.Add(Item{Kind: KindLetter, Source: "l", Title: "l", Body: "boarding letter"})
	hits, err := d.SearchContext(t.Context(), "boarding", KindLetter)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Kind != KindLetter {
		t.Fatalf("kind filter failed: %+v", hits)
	}
}
