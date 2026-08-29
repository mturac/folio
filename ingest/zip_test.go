package ingest

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/mturac/folio/store"
)

func TestWhatsAppZipFindsChatTxt(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "WA.zip")
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	w, _ := zw.Create("WhatsApp Chat/family/_chat.txt")
	w.Write([]byte("[1/1/24, 09:00:00] Ali: the Wi-Fi password is hunter2\n"))
	zw.Close()
	if err := os.WriteFile(zipPath, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	d, err := store.Open(filepath.Join(dir, "folio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })

	n, err := ImportWhatsAppPath(d, zipPath)
	if err != nil {
		t.Fatalf("import zip: %v", err)
	}
	if n != 1 {
		t.Fatalf("want 1 message, got %d", n)
	}
	hits, _ := d.Search("hunter2")
	found := false
	for _, h := range hits {
		if store.IsMsgSource(h.Source) && h.Body == "the Wi-Fi password is hunter2" {
			found = true
		}
	}
	if !found {
		t.Fatalf("zip chat must be searchable, got %+v", hits)
	}
}
