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
	if n != 2 {
		t.Fatalf("n=%d", n)
	}
	hits, err := d.Search("B12")
	if err != nil || len(hits) < 1 {
		t.Fatalf("hits=%d err=%v", len(hits), err)
	}
	found := false
	for _, h := range hits {
		if store.IsMsgSource(h.Source) && strings.Contains(h.Body, "B12") {
			found = true
			if h.When.IsZero() {
				t.Fatal("expected when from title attr")
			}
		}
	}
	if !found {
		t.Fatalf("expected message hit: %+v", hits)
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
	if n != 2 {
		t.Fatalf("n=%d", n)
	}
	hits, _ := d.Search("boarding")
	if len(hits) < 1 {
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
	if n != 3 {
		t.Fatalf("n=%d", n)
	}
}
