package ingest

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/mturac/folio/store"
)

func TestTelegramHTMLExport(t *testing.T) {
	d, err := store.Open(filepath.Join(t.TempDir(), "folio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })

	n, err := ImportTelegramHTMLPath(d, filepath.Join("..", "testdata", "chat_telegram.html"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("n=%d", n)
	}
	hits, err := d.Search("boarding")
	if err != nil || len(hits) != 1 {
		t.Fatalf("hits=%d err=%v", len(hits), err)
	}
	if !strings.Contains(hits[0].Body, "B12") {
		t.Fatalf("body=%q", hits[0].Body)
	}
	if hits[0].When.IsZero() {
		t.Fatal("expected when from title attr")
	}
}

func TestTelegramTextExport(t *testing.T) {
	d, err := store.Open(filepath.Join(t.TempDir(), "folio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })

	n, err := ImportChatPath(d, filepath.Join("..", "testdata", "chat_telegram.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("n=%d", n)
	}
	hits, _ := d.Search("boarding")
	if len(hits) != 1 {
		t.Fatalf("hits=%d", len(hits))
	}
}

func TestChatPathWhatsAppFixture(t *testing.T) {
	d, err := store.Open(filepath.Join(t.TempDir(), "folio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	n, err := ImportChatPath(d, filepath.Join("..", "testdata", "chat_whatsapp.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("n=%d", n)
	}
}
