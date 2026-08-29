package ingest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mturac/folio/store"
)

func TestSignalMarkdown(t *testing.T) {
	d, err := store.Open(filepath.Join(t.TempDir(), "folio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	n, err := ImportSignalPath(d, filepath.Join("..", "testdata", "chat_signal.md"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("n=%d", n)
	}
	hits, _ := d.Search("B12")
	found := false
	for _, h := range hits {
		if store.IsMsgSource(h.Source) && strings.Contains(h.Body, "B12") {
			found = true
		}
	}
	if !found {
		t.Fatalf("hits=%+v", hits)
	}
}

func TestSignalJSONL(t *testing.T) {
	d, err := store.Open(filepath.Join(t.TempDir(), "folio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	n, err := ImportSignalPath(d, filepath.Join("..", "testdata", "chat_signal.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("n=%d", n)
	}
	hits, _ := d.Search("boarding")
	if len(hits) < 1 {
		t.Fatal("expected hit")
	}
}

func TestPDFWithoutPopplerIndexesFilename(t *testing.T) {
	dir := t.TempDir()
	d, err := store.Open(filepath.Join(dir, "folio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	pdf := filepath.Join(dir, "boarding_pass.pdf")
	if err := os.WriteFile(pdf, []byte("%PDF-1.1 fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	n, err := ImportPDF(d, pdf)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("n=%d", n)
	}
	hits, _ := d.Search("boarding")
	if len(hits) != 1 || hits[0].Kind != store.KindPDF {
		t.Fatalf("hits=%+v", hits)
	}
}
