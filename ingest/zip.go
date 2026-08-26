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

func isOfficialChatTxt(base string) bool {
	lower := strings.ToLower(base)
	if lower == "_chat.txt" {
		return true
	}
	// iOS/desktop export: "WhatsApp Chat with Alice.txt"
	return strings.HasPrefix(lower, "whatsapp chat with ") && strings.HasSuffix(lower, ".txt")
}

func importWAZip(d *store.DB, path string) (int, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return 0, err
	}
	defer zr.Close()

	var hits []*zip.File
	for _, f := range zr.File {
		base := f.Name
		if i := strings.LastIndex(base, "/"); i >= 0 {
			base = base[i+1:]
		}
		if isOfficialChatTxt(base) {
			hits = append(hits, f)
		}
	}
	if len(hits) == 0 {
		return 0, fmt.Errorf("no _chat.txt in %s", path)
	}
	if len(hits) > 1 {
		return 0, fmt.Errorf("%s contains %d chat exports; import one zip per chat", path, len(hits))
	}

	f := hits[0]
	base := f.Name
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	rc, err := f.Open()
	if err != nil {
		return 0, err
	}
	defer rc.Close()
	source := path + "#" + SanitizeSource(base)
	return ImportWhatsApp(d, rc, source)
}

// SanitizeSource replaces ASCII control characters in a source
// identifier with spaces. Shared by zip ingest and the CLI list path.
func SanitizeSource(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, s)
}
