// folio — chats, screenshots, newsletters. Searchable. On your disk.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mturac/folio/ingest"
	"github.com/mturac/folio/store"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "folio: %v\n", err)
		os.Exit(1)
	}
}

func run(argv []string) error {
	if len(argv) < 1 {
		usage()
		return fmt.Errorf("usage")
	}
	switch argv[0] {
	case "ingest":
		return cmdIngest(argv[1:])
	case "search":
		return cmdSearch(argv[1:])
	case "list":
		return cmdList(argv[1:])
	case "serve":
		return cmdServe(argv[1:])
	default:
		usage()
		return fmt.Errorf("unknown command %q", argv[0])
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `folio — chats, screenshots, newsletters. Searchable. On your disk.

  folio ingest chat <export.txt|export.zip>   official WhatsApp/Telegram-style export
  folio ingest shots <dir>                    screenshots (tesseract OCR if installed)
  folio ingest letter <file.html|file.eml>    a newsletter
  folio search <query>                        full-text search
  folio list [chat|shot|letter]               recent items
  folio serve [:port]                         local reading room (default :8787)
`)
}

func dbPath() string {
	h, _ := os.UserHomeDir()
	dir := filepath.Join(h, ".folio")
	os.MkdirAll(dir, 0o700)
	return filepath.Join(dir, "folio.db")
}

func cmdIngest(argv []string) error {
	if len(argv) < 2 {
		usage()
		return fmt.Errorf("ingest needs kind and path")
	}
	d, err := store.Open(dbPath())
	if err != nil {
		return err
	}
	defer d.Close()
	kind, path := argv[0], argv[1]
	var n int
	switch kind {
	case "chat":
		n, err = ingest.ImportWhatsAppPath(d, path)
	case "shots":
		n, err = ingest.ImportShots(d, path, ingest.BestOCR)
	case "letter":
		f, e := os.Open(path)
		if e != nil {
			return e
		}
		defer f.Close()
		n, err = ingest.ImportLetter(d, f, path)
	default:
		usage()
		return fmt.Errorf("unknown ingest kind %q", kind)
	}
	if err != nil {
		return err
	}
	fmt.Printf("ingested %d item(s)\n", n)
	return nil
}

func cmdSearch(argv []string) error {
	if len(argv) < 1 {
		usage()
		return fmt.Errorf("search needs a query")
	}
	d, err := store.Open(dbPath())
	if err != nil {
		return err
	}
	defer d.Close()
	hits, err := d.Search(strings.Join(argv, " "))
	if err != nil {
		return err
	}
	if len(hits) == 0 {
		fmt.Println("no matches")
		return nil
	}
	for _, h := range hits {
		fmt.Printf("[%s] %s\n  %s\n", h.Kind, h.Title, clip(h.Body, 160))
	}
	return nil
}

func cmdList(argv []string) error {
	kind := ""
	if len(argv) > 0 {
		kind = argv[0]
	}
	d, err := store.Open(dbPath())
	if err != nil {
		return err
	}
	defer d.Close()
	items, err := d.List(kind, 50)
	if err != nil {
		return err
	}
	for _, it := range items {
		fmt.Printf("[%s] %s  (%s)\n", it.Kind, it.Title, ingest.SanitizeSource(it.Source))
	}
	return nil
}

func clip(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
