// Package ingest turns official exports and local files into Folio items.
// WhatsApp: only the zip/_chat.txt the user exported — we never touch
// the encrypted database.
package ingest

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/mturac/folio/store"
)

var (
	waBracket = regexp.MustCompile(`^\[(\d{1,2}/\d{1,2}/\d{2,4}),?\s+([^\]]+)\]\s+([^:]+):\s*(.*)$`)
	waDash    = regexp.MustCompile(`^(\d{1,2}/\d{1,2}/\d{2,4}),?\s+(\d{1,2}:\d{2}(?::\d{2})?(?:\s?[AP]M)?)\s+-\s+([^:]+):\s*(.*)$`)
)

// ImportWhatsApp parses an official WhatsApp export and stores it as
// one searchable chat item (source = the export path the user gave).
func ImportWhatsApp(d *store.DB, r io.Reader, source string) (int, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024)

	var b strings.Builder
	var people = map[string]struct{}{}
	msgs := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		name, text, ok := parseWALine(line)
		if !ok {
			// continuation of previous message
			if b.Len() > 0 {
				b.WriteByte('\n')
				b.WriteString(line)
			}
			continue
		}
		people[name] = struct{}{}
		fmt.Fprintf(&b, "%s: %s\n", name, text)
		msgs++
	}
	if err := sc.Err(); err != nil {
		return 0, err
	}
	if msgs == 0 {
		return 0, fmt.Errorf("no WhatsApp messages in %s", source)
	}

	title := "WhatsApp export"
	if n := len(people); n > 0 && n <= 4 {
		var names []string
		for p := range people {
			names = append(names, p)
		}
		title = strings.Join(names, ", ")
	} else if n > 4 {
		title = fmt.Sprintf("WhatsApp group (%d people)", n)
	}

	_, err := d.Add(store.Item{
		Kind:   store.KindChat,
		Source: source,
		Title:  title,
		Body:   b.String(),
	})
	if err != nil {
		return 0, err
	}
	return 1, nil
}

func parseWALine(line string) (name, text string, ok bool) {
	if m := waBracket.FindStringSubmatch(line); len(m) == 5 {
		return strings.TrimSpace(m[3]), m[4], true
	}
	if m := waDash.FindStringSubmatch(line); len(m) == 5 {
		return strings.TrimSpace(m[3]), m[4], true
	}
	return "", "", false
}
