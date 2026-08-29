// folio — chats, screenshots, newsletters. Searchable. On your disk.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mturac/folio/ingest"
	"github.com/mturac/folio/store"
)

// Set by -ldflags "-X main.version=..."
var version = "dev"

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
	case "stats":
		return cmdStats()
	case "rm":
		return cmdRm(argv[1:])
	case "watch":
		return cmdWatch(argv[1:])
	case "serve":
		return cmdServe(argv[1:])
	case "doctor":
		return cmdDoctor()
	case "export":
		return cmdExport(argv[1:])
	case "version", "-v", "--version":
		fmt.Println("folio", version)
		return nil
	default:
		usage()
		return fmt.Errorf("unknown command %q", argv[0])
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `folio — chats, screenshots, newsletters. Searchable. On your disk.

  folio ingest chat <export>                  WhatsApp zip/txt or Telegram HTML/text
  folio ingest shots <dir>                    screenshots (recursive; tesseract OCR if installed)
  folio ingest letter <file|dir>              newsletter html/eml/mbox (or a folder)
  folio watch <chat|shots|letter> <path>      re-ingest on change
  folio search <query>                        full-text search
  folio list [chat|shot|letter]               recent items
  folio stats                                 library counts
  folio export [json|md]                      dump library to stdout
  folio doctor                                check db / tesseract
  folio rm <id|source>                        remove an item
  folio serve [:port]                         local reading room (default :8787)
  folio version
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
		n, err = ingest.ImportChatPath(d, path)
	case "shots":
		n, err = ingest.ImportShots(d, path, ingest.BestOCR)
	case "letter":
		n, err = ingest.ImportLetterPath(d, path)
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

func cmdWatch(argv []string) error {
	if len(argv) < 2 {
		usage()
		return fmt.Errorf("watch needs kind and path")
	}
	kind, path := argv[0], argv[1]
	switch kind {
	case "chat", "shots", "letter":
	default:
		return fmt.Errorf("unknown watch kind %q", kind)
	}
	d, err := store.Open(dbPath())
	if err != nil {
		return err
	}
	defer d.Close()

	stop := make(chan struct{})
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigc
		close(stop)
	}()

	fmt.Printf("folio watch %s → %s (ctrl-c to stop)\n", kind, path)
	return ingest.WatchPoll([]string{path}, 2*time.Second, func(p string) (int, error) {
		return ingest.IngestWatchedPath(d, kind, p)
	}, stop, func(format string, args ...any) {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	})
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
		when := ""
		if !h.When.IsZero() {
			when = "  " + h.When.Format("2006-01-02")
		}
		fmt.Printf("[%s]%s %s\n  %s\n", h.Kind, when, h.Title, store.Snippet(h.Body, strings.Join(argv, " "), 70))
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

func cmdStats() error {
	d, err := store.Open(dbPath())
	if err != nil {
		return err
	}
	defer d.Close()
	s, err := d.Stats()
	if err != nil {
		return err
	}
	fmt.Printf("folio library: %d item(s)\n", s.Total)
	for _, k := range []string{store.KindChat, store.KindShot, store.KindLetter} {
		if n := s.ByKind[k]; n > 0 {
			fmt.Printf("  %-7s %d\n", k, n)
		}
	}
	return nil
}

func cmdRm(argv []string) error {
	if len(argv) < 1 {
		usage()
		return fmt.Errorf("rm needs id or source")
	}
	d, err := store.Open(dbPath())
	if err != nil {
		return err
	}
	defer d.Close()
	arg := argv[0]
	var ok bool
	if id, err := strconv.ParseInt(arg, 10, 64); err == nil && id > 0 {
		ok, err = d.Delete(id)
		if err != nil {
			return err
		}
	} else {
		ok, err = d.DeleteBySource(arg)
		if err != nil {
			return err
		}
	}
	if !ok {
		return fmt.Errorf("not found: %s", arg)
	}
	fmt.Println("removed")
	return nil
}

func clip(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
