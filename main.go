// folio — chats, screenshots, newsletters. Searchable. On your disk.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mturac/folio/ingest"
	"github.com/mturac/folio/store"
)

// Set by -ldflags "-X main.version=..."
var version = "0.3.0"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "folio: %v\n", err)
		os.Exit(1)
	}
}

func run(argv []string) error {
	if len(argv) < 1 {
		return cmdWelcome()
	}
	switch argv[0] {
	case "help", "-h", "--help":
		usage()
		return nil
	case "init":
		return cmdInit()
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
		return fmt.Errorf("unknown command %q — try: folio help", argv[0])
	}
}

func usage() {
	fmt.Fprint(os.Stdout, `folio — chats, screenshots, newsletters. Searchable. On your disk.

Start here:
  folio init
  folio ingest chat <WhatsApp zip or Telegram html>
  folio ingest shots <folder of screenshots>
  folio ingest letter <eml|html|mbox|folder>
  folio serve --open

Also:
  folio search <query>           full-text search
  folio list [chat|shot|letter]  recent items
  folio stats                    library counts
  folio watch <kind> <path>      re-ingest on change
  folio export [json|md]         dump library
  folio doctor                   check install
  folio rm <id|source>           remove an item
  folio version
`)
}

func cmdWelcome() error {
	fmt.Println("folio — chats, screenshots, newsletters. On your disk.")
	fmt.Println()
	n := 0
	if d, err := store.Open(dbPath()); err == nil {
		n, _ = d.Count()
		d.Close()
	}
	if n == 0 {
		fmt.Println("Your library is empty. Three steps:")
		fmt.Println()
		fmt.Println("  1. folio init")
		fmt.Println("  2. folio ingest chat ~/Downloads/WhatsApp\\ Chat.zip")
		fmt.Println("     folio ingest shots ~/Desktop/Screenshots")
		fmt.Println("  3. folio serve --open")
		fmt.Println()
		fmt.Println("Tip: drag files into the reading room once it is open.")
	} else {
		fmt.Printf("Library: %d item(s) in %s\n\n", n, dbPath())
		fmt.Println("  folio search \"boarding pass\"")
		fmt.Println("  folio serve --open")
		fmt.Println("  folio help")
	}
	return nil
}

func cmdInit() error {
	dir := folioDir()
	inbox := filepath.Join(dir, "inbox")
	if err := os.MkdirAll(inbox, 0o700); err != nil {
		return err
	}
	db := dbPath()
	d, err := store.Open(db)
	if err != nil {
		return err
	}
	n, _ := d.Count()
	d.Close()
	fmt.Printf("Ready.\n  data:  %s\n  inbox: %s\n  items: %d\n\n", db, inbox, n)
	fmt.Println("Next:")
	fmt.Println("  folio ingest chat <export>")
	fmt.Println("  folio ingest shots <folder>")
	fmt.Println("  folio serve --open")
	return nil
}

func folioDir() string {
	h, _ := os.UserHomeDir()
	dir := filepath.Join(h, ".folio")
	os.MkdirAll(dir, 0o700)
	return dir
}

func dbPath() string {
	return filepath.Join(folioDir(), "folio.db")
}

func cmdIngest(argv []string) error {
	if len(argv) < 2 {
		return fmt.Errorf("usage: folio ingest <chat|shots|letter> <path>")
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
		return fmt.Errorf("unknown kind %q (want chat, shots, or letter)", kind)
	}
	if err != nil {
		return err
	}
	fmt.Printf("ingested %d item(s)\n", n)
	return nil
}

func cmdWatch(argv []string) error {
	if len(argv) < 2 {
		return fmt.Errorf("usage: folio watch <chat|shots|letter> <path>")
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
		return fmt.Errorf("usage: folio search <query>")
	}
	d, err := store.Open(dbPath())
	if err != nil {
		return err
	}
	defer d.Close()
	q := strings.Join(argv, " ")
	hits, err := d.Search(q)
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
		fmt.Printf("[%s]%s %s\n  %s\n", h.Kind, when, h.Title, store.Snippet(h.Body, q, 70))
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
	if len(items) == 0 {
		fmt.Println("nothing yet — folio ingest …")
		return nil
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
		return fmt.Errorf("usage: folio rm <id|source>")
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

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
