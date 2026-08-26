package ingest

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/mturac/folio/store"
)

const whatsappExport = `[31/12/2023, 10:15:22] Mehmet: Can you send the boarding pass?
[31/12/2023, 10:16:01] Ayse: attached boarding-pass.pdf
[31/12/2023, 10:16:40] Mehmet: thanks — Friday flight is at 07:40
`

func TestWhatsAppExportBecomesSearchableChat(t *testing.T) {
	dir := t.TempDir()
	d, err := store.Open(filepath.Join(dir, "folio.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { d.Close() })

	n, err := ImportWhatsApp(d, strings.NewReader(whatsappExport), "family/_chat.txt")
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if n < 1 {
		t.Fatalf("want at least 1 item, got %d", n)
	}

	hits, err := d.Search("boarding")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("imported chat must be searchable for 'boarding'")
	}
	if hits[0].Kind != store.KindChat {
		t.Fatalf("kind=%s, want chat", hits[0].Kind)
	}
}
