package ingest

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"io/fs"
	"mime"
	"mime/multipart"
	"net/mail"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

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
	var when time.Time
	if isEMLSource(s) {
		msg, err := mail.ReadMessage(strings.NewReader(s))
		if err != nil {
			return 0, fmt.Errorf("eml parse failed for %s: %w", source, err)
		}
		title = strings.TrimSpace(msg.Header.Get("Subject"))
		if ds := msg.Header.Get("Date"); ds != "" {
			if t, err := mail.ParseDate(ds); err == nil {
				when = t
			}
		}
		bodySrc, err = extractMailBody(msg)
		if err != nil {
			return 0, err
		}
	}

	plain := letterPlain(bodySrc)
	if title == "" {
		title = letterTitle(bodySrc)
	}
	if title == "" {
		title = source
	}

	if _, err := d.Upsert(store.Item{
		Kind:   store.KindLetter,
		Source: source,
		Title:  title,
		Body:   plain,
		When:   when,
	}); err != nil {
		return 0, err
	}
	return 1, nil
}

func extractMailBody(msg *mail.Message) (string, error) {
	ct := msg.Header.Get("Content-Type")
	if ct == "" {
		b, err := io.ReadAll(msg.Body)
		return string(b), err
	}
	media, params, err := mime.ParseMediaType(ct)
	if err != nil {
		b, err2 := io.ReadAll(msg.Body)
		return string(b), err2
	}
	if strings.HasPrefix(media, "multipart/") {
		return readMultipart(msg.Body, params["boundary"])
	}
	b, err := io.ReadAll(msg.Body)
	if err != nil {
		return "", err
	}
	return decodePart(string(b), msg.Header.Get("Content-Transfer-Encoding"), media), nil
}

func readMultipart(r io.Reader, boundary string) (string, error) {
	if boundary == "" {
		b, err := io.ReadAll(r)
		return string(b), err
	}
	mr := multipart.NewReader(r, boundary)
	var htmlPart, textPart string
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		ct := p.Header.Get("Content-Type")
		media, _, _ := mime.ParseMediaType(ct)
		raw, err := io.ReadAll(p)
		if err != nil {
			return "", err
		}
		body := decodePart(string(raw), p.Header.Get("Content-Transfer-Encoding"), media)
		switch {
		case strings.HasPrefix(media, "text/html"):
			htmlPart = body
		case strings.HasPrefix(media, "text/plain") && textPart == "":
			textPart = body
		case strings.HasPrefix(media, "multipart/"):
			_, params, _ := mime.ParseMediaType(ct)
			nested, err := readMultipart(bytes.NewReader(raw), params["boundary"])
			if err != nil {
				return "", err
			}
			if htmlPart == "" && strings.Contains(strings.ToLower(nested), "<html") {
				htmlPart = nested
			} else if textPart == "" {
				textPart = nested
			}
		}
	}
	if htmlPart != "" {
		return htmlPart, nil
	}
	return textPart, nil
}

func decodePart(s, transfer, media string) string {
	switch strings.ToLower(strings.TrimSpace(transfer)) {
	case "base64":
		cleaned := strings.Map(func(r rune) rune {
			if r == '\r' || r == '\n' || r == ' ' || r == '\t' {
				return -1
			}
			return r
		}, s)
		if decoded, err := base64.StdEncoding.DecodeString(cleaned); err == nil {
			s = string(decoded)
		}
	case "quoted-printable":
		// net/mail already decodes quoted-printable for single parts
		// in some paths; leave as-is for multipart raw reads.
	}
	_ = media
	return s
}

func isEMLSource(s string) bool {
	head := s
	if len(head) > 800 {
		head = head[:800]
	}
	low := strings.ToLower(head)
	hasSubject := strings.Contains(low, "\nsubject:") || strings.HasPrefix(low, "subject:")
	hasFrom := strings.Contains(low, "\nfrom:") || strings.HasPrefix(low, "from:")
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

// ImportLetterPath accepts a single .html/.eml/.mbox file or a directory of them.
func ImportLetterPath(d *store.DB, path string) (int, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	if info.IsDir() {
		n := 0
		err := filepath.WalkDir(path, func(p string, e fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if e.IsDir() {
				return nil
			}
			if !isLetterFile(p) {
				return nil
			}
			c, err := importLetterFile(d, p)
			if err != nil {
				fmt.Fprintf(os.Stderr, "folio: letter %s: %v\n", p, err)
				return nil
			}
			n += c
			return nil
		})
		return n, err
	}
	return importLetterFile(d, path)
}

func isLetterFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".html") ||
		strings.HasSuffix(lower, ".htm") ||
		strings.HasSuffix(lower, ".eml") ||
		strings.HasSuffix(lower, ".mbox")
}

func importLetterFile(d *store.DB, path string) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".mbox") || looksLikeMBOX(string(raw)) {
		return ImportMBOX(d, string(raw), path)
	}
	return ImportLetter(d, strings.NewReader(string(raw)), path)
}

func looksLikeMBOX(s string) bool {
	if strings.HasPrefix(s, "From ") {
		return strings.Count(s, "\nFrom ") >= 1
	}
	return false
}

// ImportMBOX splits a Unix mbox and upserts each message as its own letter.
func ImportMBOX(d *store.DB, raw, source string) (int, error) {
	msgs := splitMBOX(raw)
	if len(msgs) == 0 {
		return 0, fmt.Errorf("no messages in mbox %s", source)
	}
	n := 0
	for i, msg := range msgs {
		src := fmt.Sprintf("%s#%d", source, i+1)
		c, err := ImportLetter(d, strings.NewReader(msg), src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "folio: mbox message %d in %s: %v\n", i+1, source, err)
			continue
		}
		n += c
	}
	if n == 0 {
		return 0, fmt.Errorf("no messages ingested from mbox %s", source)
	}
	return n, nil
}

func splitMBOX(raw string) []string {
	lines := strings.Split(raw, "\n")
	var msgs []string
	var b strings.Builder
	flush := func() {
		s := strings.TrimSpace(b.String())
		if s != "" {
			msgs = append(msgs, s)
		}
		b.Reset()
	}
	for i, line := range lines {
		if strings.HasPrefix(line, "From ") && i > 0 {
			flush()
			continue // skip envelope From line for subsequent messages
		}
		if i == 0 && strings.HasPrefix(line, "From ") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	flush()
	return msgs
}
