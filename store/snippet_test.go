package store

import (
	"strings"
	"testing"
)

func TestSnippetAroundHit(t *testing.T) {
	body := strings.Repeat("x", 100) + " boarding pass " + strings.Repeat("y", 100)
	got := Snippet(body, "boarding", 20)
	if !strings.Contains(got, "boarding") {
		t.Fatalf("got %q", got)
	}
	if !strings.HasPrefix(got, "…") {
		t.Fatalf("expected leading ellipsis, got %q", got)
	}
}

func TestSnippetFallback(t *testing.T) {
	got := Snippet("short body", "zzzz", 10)
	if got != "short body" {
		t.Fatalf("got %q", got)
	}
}
