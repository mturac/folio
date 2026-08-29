package ingest

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	if n != 3 {
		t.Fatalf("want 3 messages, got %d", n)
	}

	hits, err := d.Search("boarding")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("imported chat must be searchable for 'boarding'")
	}
	// Prefer a per-message hit with speaker title.
	foundMsg := false
	for _, h := range hits {
		if store.IsMsgSource(h.Source) && h.Title == "Mehmet" && strings.Contains(h.Body, "boarding") {
			foundMsg = true
			if h.When.IsZero() || h.When.Year() != 2023 {
				t.Fatalf("message when=%v", h.When)
			}
			break
		}
	}
	if !foundMsg {
		t.Fatalf("expected per-message hit, got %+v", hits)
	}

	s, err := d.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if s.ByKind[store.KindChat] != 1 {
		t.Fatalf("stats should count 1 thread, got %d", s.ByKind[store.KindChat])
	}
}

func TestWhatsAppReingestUpdatesBody(t *testing.T) {
	d, err := store.Open(filepath.Join(t.TempDir(), "folio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })

	_, err = ImportWhatsApp(d, strings.NewReader(whatsappExport), "family/_chat.txt")
	if err != nil {
		t.Fatal(err)
	}
	updated := whatsappExport + "[31/12/2023, 11:00:00] Mehmet: gate changed to B12\n"
	n, err := ImportWhatsApp(d, strings.NewReader(updated), "family/_chat.txt")
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 {
		t.Fatalf("want 4 messages after reingest, got %d", n)
	}
	hits, err := d.Search("B12")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) < 1 {
		t.Fatal("reingest must refresh FTS")
	}
	total, _ := d.Count()
	if total != 5 { // 1 thread + 4 messages
		t.Fatalf("want 5 rows, got %d", total)
	}
}

func TestParseWAStamp(t *testing.T) {
	got := parseWAStamp("31/12/2023", "10:15:22")
	if got.IsZero() {
		t.Fatal("expected parsed time")
	}
	if got.Month() != time.December && got.Day() != 31 {
		t.Logf("parsed=%v", got)
	}
}
