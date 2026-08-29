package ingest

import (
	"os"
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
Date: Tue, 31 Dec 2023 09:00:00 +0000
Content-Type: text/html; charset=utf-8

` + letterHTML

const letterMultipart = `From: editor@dispatch.example
Subject: Multipart Dispatch
Date: Wed, 01 Jan 2024 12:00:00 +0000
MIME-Version: 1.0
Content-Type: multipart/alternative; boundary="bound42"

--bound42
Content-Type: text/plain; charset=utf-8

plain only should lose to html
--bound42
Content-Type: text/html; charset=utf-8

<html><body><p>Searchable boarding pass tip inside html part</p></body></html>
--bound42--
`

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
	if hits[0].When.IsZero() {
		t.Fatal("eml Date header should set When")
	}
}

func TestNewsletterMultipartPrefersHTML(t *testing.T) {
	d, err := store.Open(filepath.Join(t.TempDir(), "folio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })

	_, err = ImportLetter(d, strings.NewReader(letterMultipart), "multi.eml")
	if err != nil {
		t.Fatal(err)
	}
	hits, err := d.Search("boarding")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("html part must be indexed, got %d", len(hits))
	}
	if strings.Contains(hits[0].Body, "plain only") {
		t.Fatal("should prefer html over plain")
	}
}

func TestMBOXSplitsMessages(t *testing.T) {
	d, err := store.Open(filepath.Join(t.TempDir(), "folio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })

	mbox := `From a@b c
From: a@b.c
Subject: One
Date: Tue, 31 Dec 2023 09:00:00 +0000
Content-Type: text/plain

alpha boarding

From b@c d
From: b@c.d
Subject: Two
Date: Wed, 01 Jan 2024 09:00:00 +0000
Content-Type: text/plain

beta wifi
`
	n, err := ImportMBOX(d, mbox, "box.mbox")
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("n=%d", n)
	}
	hits, _ := d.Search("wifi")
	if len(hits) != 1 || hits[0].Title != "Two" {
		t.Fatalf("hits=%+v", hits)
	}
}

func TestLetterPathDirectory(t *testing.T) {
	dir := t.TempDir()
	d, err := store.Open(filepath.Join(dir, "folio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	letters := filepath.Join(dir, "letters")
	os.MkdirAll(letters, 0o755)
	os.WriteFile(filepath.Join(letters, "a.eml"), []byte(letterEML), 0o644)
	os.WriteFile(filepath.Join(letters, "skip.txt"), []byte("nope"), 0o644)
	n, err := ImportLetterPath(d, letters)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("n=%d", n)
	}
}
