package store

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchFTSOperatorsDoNotError(t *testing.T) {
	d, err := Open(filepath.Join(t.TempDir(), "folio.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { d.Close() })

	_, err = d.Add(Item{Kind: KindChat, Source: "a", Title: "apple pie", Body: "apple pie recipe"})
	if err != nil {
		t.Fatal(err)
	}

	for _, q := range []string{"apple OR)", "NEAR(", `")`, "AND OR NOT"} {
		hits, err := d.Search(q)
		if err != nil {
			t.Fatalf("Search(%q) must not return FTS syntax error: %v", q, err)
		}
		_ = hits
	}

	hits, err := d.Search("apple")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("literal search still works, got %d", len(hits))
	}
}

func TestSearchEmptySliceNotNil(t *testing.T) {
	d, err := Open(filepath.Join(t.TempDir(), "folio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	hits, err := d.Search("zzzz-no-such")
	if err != nil {
		t.Fatal(err)
	}
	if hits == nil {
		t.Fatal("empty result must be non-nil slice for JSON []")
	}
}

func TestSanitizeFTS5QuotesTokens(t *testing.T) {
	got := sanitizeFTS5(`apple OR)`)
	if strings.Contains(got, "OR)") && !strings.Contains(got, `"`) {
		t.Fatalf("expected quoted tokens, got %q", got)
	}
}
