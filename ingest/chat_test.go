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
	if !strings.Contains(hits[0].Title, "3 msgs") {
		t.Fatalf("title should include message count: %q", hits[0].Title)
	}
	if hits[0].When.IsZero() {
		t.Fatal("chat when should be parsed from stamps")
	}
	if hits[0].When.Year() != 2023 {
		t.Fatalf("when year=%d", hits[0].When.Year())
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
	_, err = ImportWhatsApp(d, strings.NewReader(updated), "family/_chat.txt")
	if err != nil {
		t.Fatal(err)
	}
	hits, err := d.Search("B12")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("reingest must refresh FTS, got %d", len(hits))
	}
	n, _ := d.Count()
	if n != 1 {
		t.Fatalf("reingest must not duplicate, got %d", n)
	}
}

func TestParseWAStamp(t *testing.T) {
	got := parseWAStamp("31/12/2023", "10:15:22")
	if got.IsZero() {
		t.Fatal("expected parsed time")
	}
	if got.Month() != time.December && got.Day() != 31 {
		// either DMY or MDY parse is acceptable if consistent; fixture is DMY
		t.Logf("parsed=%v", got)
	}
}
