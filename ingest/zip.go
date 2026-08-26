package ingest

import (
	"archive/zip"
	"fmt"
	"os"
	"strings"

	"github.com/mturac/folio/store"
)

// ImportWhatsAppPath accepts a _chat.txt or a zip that contains one.
func ImportWhatsAppPath(d *store.DB, path string) (int, error) {
	if strings.HasSuffix(strings.ToLower(path), ".zip") {
		return importWAZip(d, path)
	}
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return ImportWhatsApp(d, f, path)
}

func importWAZip(d *store.DB, path string) (int, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return 0, err
	}
	defer zr.Close()

	total := 0
	found := false
	for _, f := range zr.File {
		name := f.Name
		base := name
		if i := strings.LastIndex(name, "/"); i >= 0 {
			base = name[i+1:]
		}
		if !strings.HasSuffix(strings.ToLower(base), ".txt") {
			continue
		}
		if !strings.Contains(strings.ToLower(base), "chat") && base != "_chat.txt" {
			// WhatsApp names the file "_chat.txt" or "WhatsApp Chat with X.txt"
			if !strings.Contains(strings.ToLower(name), "whatsapp") {
				continue
			}
		}
		rc, err := f.Open()
		if err != nil {
			return total, err
		}
		n, err := ImportWhatsApp(d, rc, path+"#"+name)
		rc.Close()
		if err != nil {
			return total, err
		}
		total += n
		found = true
	}
	if !found {
		return 0, fmt.Errorf("no _chat.txt in %s", path)
	}
	return total, nil
}
