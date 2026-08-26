package ingest

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/mturac/folio/store"
)

func writeZip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestZipIgnoresUnrelatedChatNamedFiles(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "WA.zip")
	writeZip(t, zipPath, map[string]string{
		"WhatsApp Chat/family/_chat.txt":           "[1/1/24, 09:00:00] Ali: the Wi-Fi password is hunter2\n",
		"WhatsApp Chat/family/notes_about_chat.txt": "this is not a WhatsApp export\n",
	})

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
		t.Fatalf("want exactly 1 chat item, got %d", n)
	}
	c, _ := d.Count()
	if c != 1 {
		t.Fatalf("unrelated notes_about_chat.txt must not be ingested, count=%d", c)
	}
}
