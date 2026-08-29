package ingest

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mturac/folio/store"
)

// Signal Desktop / signal-export markdown lines:
// [2019-05-29, 15:04] Me: How is everyone?
var signalMD = regexp.MustCompile(`^\[(\d{4}-\d{2}-\d{2}),?\s+(\d{1,2}:\d{2}(?::\d{2})?)\]\s+([^:]+):\s*(.*)$`)

// ImportSignalMarkdown parses signal-export style markdown chat files.
func ImportSignalMarkdown(d *store.DB, r io.Reader, source string) (int, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
	var lines []chatLine
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		m := signalMD.FindStringSubmatch(line)
		if m == nil {
			if len(lines) > 0 && !strings.HasPrefix(line, "![") {
				prev := &lines[len(lines)-1]
				prev.text += "\n" + line
			}
			continue
		}
		when := parseSignalStamp(m[1], m[2])
		lines = append(lines, chatLine{name: strings.TrimSpace(m[3]), text: m[4], when: when})
	}
	if err := sc.Err(); err != nil {
		return 0, err
	}
	if len(lines) == 0 {
		return 0, fmt.Errorf("no Signal messages in %s", source)
	}
	return storeChatLines(d, source, "Signal", lines)
}

func parseSignalStamp(date, clock string) time.Time {
	raw := strings.TrimSpace(date) + " " + strings.TrimSpace(clock)
	for _, layout := range []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
	} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t
		}
	}
	return time.Time{}
}

// ImportSignalJSONL reads Signal Desktop chat-history JSONL (one JSON object per line).
// Best-effort: looks for body/text/message + timestamp/sent_at + sender/sourceName.
func ImportSignalJSONL(d *store.DB, r io.Reader, source string) (int, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
	var lines []chatLine
	for sc.Scan() {
		raw := strings.TrimSpace(sc.Text())
		if raw == "" {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(raw), &obj); err != nil {
			continue
		}
		text := firstString(obj, "body", "text", "message", "content")
		if text == "" {
			continue
		}
		name := firstString(obj, "sourceName", "sender", "author", "from", "profileName", "name")
		if name == "" {
			name = "Signal"
		}
		when := time.Time{}
		if ts, ok := obj["timestamp"].(float64); ok && ts > 0 {
			// ms or seconds
			if ts > 1e12 {
				when = time.UnixMilli(int64(ts))
			} else {
				when = time.Unix(int64(ts), 0)
			}
		} else if s := firstString(obj, "sent_at", "sentAt", "date", "timestamp"); s != "" {
			when = parseLooseTime(s)
		}
		lines = append(lines, chatLine{name: name, text: text, when: when})
	}
	if err := sc.Err(); err != nil {
		return 0, err
	}
	if len(lines) == 0 {
		return 0, fmt.Errorf("no Signal JSONL messages in %s", source)
	}
	return storeChatLines(d, source, "Signal", lines)
}

func firstString(obj map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := obj[k]; ok {
			switch t := v.(type) {
			case string:
				if strings.TrimSpace(t) != "" {
					return strings.TrimSpace(t)
				}
			}
		}
	}
	return ""
}

func parseLooseTime(s string) time.Time {
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func looksLikeSignalMarkdown(s string) bool {
	n, hits := 0, 0
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		n++
		if signalMD.MatchString(line) {
			hits++
		}
		if n >= 20 {
			break
		}
	}
	return hits > 0 && hits*2 >= n
}

func looksLikeJSONL(s string) bool {
	line := strings.TrimSpace(strings.SplitN(s, "\n", 2)[0])
	return strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}")
}

// ImportSignalPath loads a Signal markdown or JSONL export.
func ImportSignalPath(d *store.DB, path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, 12<<20))
	if err != nil {
		return 0, err
	}
	s := string(raw)
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".jsonl") || strings.HasSuffix(lower, ".json") || looksLikeJSONL(s) {
		return ImportSignalJSONL(d, strings.NewReader(s), path)
	}
	return ImportSignalMarkdown(d, strings.NewReader(s), path)
}

func isSignalNamed(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return strings.Contains(base, "signal") || strings.HasSuffix(base, ".jsonl")
}
