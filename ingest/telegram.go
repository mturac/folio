package ingest

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/mturac/folio/store"
)

var (
	tgTextLine = regexp.MustCompile(`^(\d{1,2}\.\d{1,2}\.\d{2,4})\s+(\d{1,2}:\d{2}(?::\d{2})?):\s+([^:]+):\s*(.*)$`)
	tgFromName = regexp.MustCompile(`(?is)<div class="from_name">\s*([^<]+?)\s*</div>`)
	tgText     = regexp.MustCompile(`(?is)<div class="text">\s*(.*?)\s*</div>`)
	tgDate     = regexp.MustCompile(`(?is)<div class="[^"]*date[^"]*"[^>]*title="([^"]+)"`)
	tgMessage  = regexp.MustCompile(`(?is)<div class="message[^"]*"[^>]*>.*?</div>\s*</div>`)
)

// ImportChatPath auto-detects WhatsApp vs Telegram vs Signal exports.
func ImportChatPath(d *store.DB, path string) (int, error) {
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".zip") {
		return importWAZip(d, path)
	}
	if strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".htm") {
		return ImportTelegramHTMLPath(d, path)
	}
	if strings.HasSuffix(lower, ".jsonl") || strings.HasSuffix(lower, ".json") || isSignalNamed(path) {
		return ImportSignalPath(d, path)
	}
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, 8<<20))
	if err != nil {
		return 0, err
	}
	s := string(raw)
	if looksLikeJSONL(s) {
		return ImportSignalJSONL(d, strings.NewReader(s), path)
	}
	if looksLikeSignalMarkdown(s) {
		return ImportSignalMarkdown(d, strings.NewReader(s), path)
	}
	if looksLikeTelegramText(s) {
		return ImportTelegramText(d, strings.NewReader(s), path)
	}
	return ImportWhatsApp(d, strings.NewReader(s), path)
}

func looksLikeTelegramText(s string) bool {
	lines := 0
	hits := 0
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines++
		if tgTextLine.MatchString(line) {
			hits++
		}
		if lines >= 20 {
			break
		}
	}
	return hits > 0 && hits*2 >= lines
}

// ImportTelegramText parses Telegram-style "DD.MM.YYYY HH:MM:SS: Name: text" lines.
func ImportTelegramText(d *store.DB, r io.Reader, source string) (int, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	var lines []chatLine
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := tgTextLine.FindStringSubmatch(line)
		if m == nil {
			if len(lines) > 0 {
				prev := &lines[len(lines)-1]
				prev.text += "\n" + line
			}
			continue
		}
		lines = append(lines, chatLine{
			name: strings.TrimSpace(m[3]),
			text: m[4],
			when: parseTGStamp(m[1], m[2]),
		})
	}
	if len(lines) == 0 {
		return 0, fmt.Errorf("no Telegram messages in %s", source)
	}
	return storeChatLines(d, source, "Telegram", lines)
}

// ImportTelegramHTMLPath reads a Telegram Desktop HTML export.
func ImportTelegramHTMLPath(d *store.DB, path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return ImportTelegramHTML(d, f, path)
}

func ImportTelegramHTML(d *store.DB, r io.Reader, source string) (int, error) {
	raw, err := io.ReadAll(io.LimitReader(r, 12<<20))
	if err != nil {
		return 0, err
	}
	htmlSrc := string(raw)
	blocks := tgMessage.FindAllString(htmlSrc, -1)
	if len(blocks) == 0 {
		return importTelegramHTMLLoose(d, htmlSrc, source)
	}
	var lines []chatLine
	for _, block := range blocks {
		name := ""
		if m := tgFromName.FindStringSubmatch(block); len(m) == 2 {
			name = strings.TrimSpace(htmlUnescapeLite(stripTags(m[1])))
		}
		text := ""
		if m := tgText.FindStringSubmatch(block); len(m) == 2 {
			text = strings.TrimSpace(letterPlain(m[1]))
		}
		if name == "" || text == "" {
			continue
		}
		when := time.Time{}
		if m := tgDate.FindStringSubmatch(block); len(m) == 2 {
			when = parseTGTitleDate(m[1])
		}
		lines = append(lines, chatLine{name: name, text: text, when: when})
	}
	if len(lines) == 0 {
		return 0, fmt.Errorf("no Telegram HTML messages in %s", source)
	}
	return storeChatLines(d, source, "Telegram", lines)
}

func importTelegramHTMLLoose(d *store.DB, htmlSrc, source string) (int, error) {
	names := tgFromName.FindAllStringSubmatch(htmlSrc, -1)
	texts := tgText.FindAllStringSubmatch(htmlSrc, -1)
	if len(names) == 0 || len(texts) == 0 {
		return 0, fmt.Errorf("no Telegram HTML messages in %s", source)
	}
	n := len(names)
	if len(texts) < n {
		n = len(texts)
	}
	var lines []chatLine
	for i := 0; i < n; i++ {
		name := strings.TrimSpace(htmlUnescapeLite(stripTags(names[i][1])))
		text := strings.TrimSpace(letterPlain(texts[i][1]))
		if name == "" || text == "" {
			continue
		}
		lines = append(lines, chatLine{name: name, text: text})
	}
	if len(lines) == 0 {
		return 0, fmt.Errorf("no Telegram HTML messages in %s", source)
	}
	return storeChatLines(d, source, "Telegram", lines)
}

func parseTGStamp(date, clock string) time.Time {
	raw := strings.TrimSpace(date) + " " + strings.TrimSpace(clock)
	for _, layout := range []string{
		"2.1.2006 15:04:05",
		"2.1.06 15:04:05",
		"2.1.2006 15:04",
		"2.1.06 15:04",
		"02.01.2006 15:04:05",
		"02.01.06 15:04:05",
	} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t
		}
	}
	return time.Time{}
}

func parseTGTitleDate(title string) time.Time {
	title = strings.TrimSpace(title)
	if i := strings.Index(title, " UTC"); i > 0 {
		title = title[:i]
	}
	parts := strings.Fields(title)
	if len(parts) >= 2 {
		return parseTGStamp(parts[0], parts[1])
	}
	return time.Time{}
}

func htmlUnescapeLite(s string) string {
	return strings.NewReplacer(
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&quot;", `"`,
		"&#39;", "'",
		"&nbsp;", " ",
	).Replace(s)
}
