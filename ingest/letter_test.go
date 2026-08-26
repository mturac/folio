package ingest

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/mturac/folio/store"
)

const letterHTML = `<!doctype html><html><head>
<title>The Weekly Dispatch #42</title></head><body>
<h1>The Weekly Dispatch #42</h1>
<p>This week: why boarding passes still live in screenshots, and how
to keep newsletters out of your inbox.</p>
<script>track('open')</script>
</body></html>`

const letterEML = `From: editor@dispatch.example
Subject: The Weekly Dispatch #42
Content-Type: text/html; charset=utf-8

` + letterHTML

func TestNewsletterHTMLIsSearchable(t *testing.T) {
	d, err := store.Open(filepath.Join(t.TempDir(), "folio.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { d.Close() })

	n, err := ImportLetter(d, strings.NewReader(letterHTML), "dispatch-42.html")
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if n != 1 {
		t.Fatalf("want 1, got %d", n)
	}
	hits, err := d.Search("boarding")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 1 || hits[0].Kind != store.KindLetter {
		t.Fatalf("letter body must be searchable: %+v", hits)
	}
	if strings.Contains(hits[0].Body, "track(") {
		t.Fatal("script must not leak into letter body")
	}
}

func TestNewsletterEMLUsesSubject(t *testing.T) {
	d, err := store.Open(filepath.Join(t.TempDir(), "folio.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { d.Close() })

	_, err = ImportLetter(d, strings.NewReader(letterEML), "dispatch-42.eml")
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	hits, _ := d.Search("Weekly")
	if len(hits) != 1 {
		t.Fatalf("want 1, got %d", len(hits))
	}
	if !strings.Contains(hits[0].Title, "Weekly Dispatch") {
		t.Fatalf("title should be Subject, got %q", hits[0].Title)
	}
}
