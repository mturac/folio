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
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "ingest":
		cmdIngest(os.Args[2:])
	case "search":
		cmdSearch(os.Args[2:])
	case "list":
		cmdList(os.Args[2:])
	case "serve":
		cmdServe(os.Args[2:])
	default:
		usage()
		os.Exit(2)
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

func openDB() *store.DB {
	d, err := store.Open(dbPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "folio: %v\n", err)
		os.Exit(1)
	}
	return d
}

func cmdIngest(argv []string) {
	if len(argv) < 2 {
		usage()
		os.Exit(2)
	}
	d := openDB()
	defer d.Close()
	kind, path := argv[0], argv[1]
	var n int
	var err error
	switch kind {
	case "chat":
		n, err = ingest.ImportWhatsAppPath(d, path)
	case "shots":
		n, err = ingest.ImportShots(d, path, ingest.BestOCR)
	case "letter":
		f, e := os.Open(path)
		if e != nil {
			fmt.Fprintf(os.Stderr, "folio: %v\n", e)
			os.Exit(1)
		}
		defer f.Close()
		n, err = ingest.ImportLetter(d, f, path)
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "folio: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("ingested %d item(s)\n", n)
}

func cmdSearch(argv []string) {
	if len(argv) < 1 {
		usage()
		os.Exit(2)
	}
	d := openDB()
	defer d.Close()
	hits, err := d.Search(strings.Join(argv, " "))
	if err != nil {
		fmt.Fprintf(os.Stderr, "folio: %v\n", err)
		os.Exit(1)
	}
	if len(hits) == 0 {
		fmt.Println("no matches")
		return
	}
	for _, h := range hits {
		fmt.Printf("[%s] %s\n  %s\n", h.Kind, h.Title, clip(h.Body, 160))
	}
}

func cmdList(argv []string) {
	kind := ""
	if len(argv) > 0 {
		kind = argv[0]
	}
	d := openDB()
	defer d.Close()
	items, err := d.List(kind, 50)
	if err != nil {
		fmt.Fprintf(os.Stderr, "folio: %v\n", err)
		os.Exit(1)
	}
	for _, it := range items {
		fmt.Printf("[%s] %s  (%s)\n", it.Kind, it.Title, it.Source)
	}
}

func clip(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
