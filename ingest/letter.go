package ingest

import (
	"fmt"
	"html"
	"io"
	"net/mail"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mturac/folio/store"
)

var (
	scriptBlock = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>|<style[^>]*>.*?</style>`)
	tagRe       = regexp.MustCompile(`(?s)<[^>]+>`)
	titleRe     = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	h1Re        = regexp.MustCompile(`(?is)<h1[^>]*>(.*?)</h1>`)
	multiWS     = regexp.MustCompile(`[\s\x{00a0}\x{202f}]+`)
)

// ImportLetter stores a newsletter from HTML or .eml bytes.
func ImportLetter(d *store.DB, r io.Reader, source string) (int, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	s := string(raw)

	title := ""
	bodySrc := s
	if isEMLSource(source, s) {
		msg, err := mail.ReadMessage(strings.NewReader(s))
		if err != nil {
			return 0, fmt.Errorf("eml parse failed for %s: %w", source, err)
		}
		title = strings.TrimSpace(msg.Header.Get("Subject"))
		b, err := io.ReadAll(msg.Body)
		if err != nil {
			return 0, err
		}
		bodySrc = string(b)
	}

	plain := letterPlain(bodySrc)
	if title == "" {
		title = letterTitle(bodySrc)
	}
	if title == "" {
		title = source
	}

	if _, err := d.Add(store.Item{
		Kind:   store.KindLetter,
		Source: source,
		Title:  title,
		Body:   plain,
	}); err != nil {
		return 0, err
	}
	return 1, nil
}

func isEMLSource(source, s string) bool {
	ext := strings.ToLower(filepath.Ext(source))
	if ext == ".eml" {
		return true
	}
	head := s
	if len(head) > 800 {
		head = head[:800]
	}
	hasSubject := strings.Contains(head, "\nSubject:") || strings.HasPrefix(head, "Subject:")
	hasFrom := strings.Contains(head, "\nFrom:") || strings.HasPrefix(head, "From:")
	return hasSubject && hasFrom
}

func letterTitle(htmlSrc string) string {
	if m := titleRe.FindStringSubmatch(htmlSrc); len(m) == 2 {
		return strings.TrimSpace(html.UnescapeString(stripTags(m[1])))
	}
	if m := h1Re.FindStringSubmatch(htmlSrc); len(m) == 2 {
		return strings.TrimSpace(html.UnescapeString(stripTags(m[1])))
	}
	return ""
}

func letterPlain(htmlSrc string) string {
	s := scriptBlock.ReplaceAllString(htmlSrc, " ")
	s = html.UnescapeString(stripTags(s))
	s = strings.NewReplacer("\u00a0", " ", "\u202f", " ", "\u200b", "").Replace(s)
	return strings.TrimSpace(multiWS.ReplaceAllString(s, " "))
}

func stripTags(s string) string {
	return tagRe.ReplaceAllString(s, " ")
}
