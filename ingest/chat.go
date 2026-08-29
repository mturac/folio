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
	"time"

	"github.com/mturac/folio/store"
)

var (
	waBracket = regexp.MustCompile(`^\[(\d{1,2}/\d{1,2}/\d{2,4}),?\s+([^\]]+)\]\s+([^:]+):\s*(.*)$`)
	// Comma after the date is required (official export format). A
	// user-authored line like "12/31/23 10:15 - wait: no" is treated
	// as a continuation, not a new message.
	waDash = regexp.MustCompile(`^(\d{1,2}/\d{1,2}/\d{2,4}),\s+(\d{1,2}:\d{2}(?::\d{2})?(?:\s?[AP]M)?)\s+-\s+([^:]+):\s*(.*)$`)
)

const maxChatMessages = 200000

type chatLine struct {
	name string
	text string
	when time.Time
}

// ImportWhatsApp parses an official WhatsApp export into one thread
// summary plus one searchable row per message.
func ImportWhatsApp(d *store.DB, r io.Reader, source string) (int, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024)

	var lines []chatLine
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		name, text, when, ok := parseWALine(line)
		if !ok {
			if len(lines) == 0 {
				continue
			}
			prev := &lines[len(lines)-1]
			if len(prev.text)+1+len(line) > 64<<10 {
				continue
			}
			prev.text += "\n" + line
			continue
		}
		if len(lines) >= maxChatMessages {
			return 0, fmt.Errorf("chat export %s exceeds %d messages; split the export", source, maxChatMessages)
		}
		lines = append(lines, chatLine{name: name, text: text, when: when})
	}
	if err := sc.Err(); err != nil {
		return 0, err
	}
	if len(lines) == 0 {
		return 0, fmt.Errorf("no WhatsApp messages in %s", source)
	}
	return storeChatLines(d, source, "WhatsApp", lines)
}

func storeChatLines(d *store.DB, source, brand string, lines []chatLine) (int, error) {
	people := map[string]struct{}{}
	var firstWhen, lastWhen time.Time
	msgs := make([]store.Item, 0, len(lines))
	for i, ln := range lines {
		people[ln.name] = struct{}{}
		if !ln.when.IsZero() {
			if firstWhen.IsZero() || ln.when.Before(firstWhen) {
				firstWhen = ln.when
			}
			if lastWhen.IsZero() || ln.when.After(lastWhen) {
				lastWhen = ln.when
			}
		}
		msgs = append(msgs, store.Item{
			Kind:   store.KindChat,
			Source: store.MsgSource(source, i+1),
			Title:  ln.name,
			Body:   ln.text,
			When:   ln.when,
		})
	}

	title := brand + " export"
	if n := len(people); n > 0 && n <= 4 {
		var names []string
		for p := range people {
			names = append(names, p)
		}
		title = strings.Join(names, ", ")
	} else if n > 4 {
		title = fmt.Sprintf("%s group (%d people)", brand, n)
	}
	title = fmt.Sprintf("%s · %d msgs", title, len(lines))

	when := lastWhen
	if when.IsZero() {
		when = firstWhen
	}

	n, err := d.ReplaceChat(store.Item{
		Kind:   store.KindChat,
		Source: source,
		Title:  title,
		Body:   "", // per-message rows carry FTS text; thread is for list/stats
		When:   when,
	}, msgs)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func parseWALine(line string) (name, text string, when time.Time, ok bool) {
	if m := waBracket.FindStringSubmatch(line); len(m) == 5 {
		return strings.TrimSpace(m[3]), m[4], parseWAStamp(m[1], m[2]), true
	}
	if m := waDash.FindStringSubmatch(line); len(m) == 5 {
		return strings.TrimSpace(m[3]), m[4], parseWAStamp(m[1], m[2]), true
	}
	return "", "", time.Time{}, false
}

func parseWAStamp(date, clock string) time.Time {
	date = strings.TrimSpace(date)
	clock = strings.TrimSpace(clock)
	candidates := []string{
		"1/2/2006 15:04:05",
		"1/2/06 15:04:05",
		"1/2/2006 15:04",
		"1/2/06 15:04",
		"1/2/2006 3:04:05 PM",
		"1/2/06 3:04:05 PM",
		"1/2/2006 3:04 PM",
		"1/2/06 3:04 PM",
		"02/01/2006 15:04:05",
		"02/01/06 15:04:05",
		"2/1/2006 15:04:05",
		"2/1/06 15:04:05",
	}
	raw := date + " " + clock
	for _, layout := range candidates {
		if t, err := time.Parse(layout, raw); err == nil {
			return t
		}
	}
	return time.Time{}
}
