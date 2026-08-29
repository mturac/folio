package ingest

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mturac/folio/store"
)

func TestWatchPollIngestsThenStops(t *testing.T) {
	dir := t.TempDir()
	d, err := store.Open(filepath.Join(dir, "folio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })

	src := filepath.Join(dir, "note.eml")
	body := "From: a@b.c\nSubject: hi\nDate: Tue, 31 Dec 2023 09:00:00 +0000\nContent-Type: text/plain\n\nboarding watch test\n"
	if err := os.WriteFile(src, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	stop := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- WatchPoll([]string{src}, 50*time.Millisecond, func(path string) (int, error) {
			return IngestWatchedPath(d, "letter", path)
		}, stop, nil)
	}()

	deadline := time.After(2 * time.Second)
	for {
		n, _ := d.Count()
		if n >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("watch did not ingest in time")
		case <-time.After(20 * time.Millisecond):
		}
	}
	close(stop)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("watch did not stop")
	}
	hits, err := d.Search("boarding")
	if err != nil || len(hits) != 1 {
		t.Fatalf("hits=%d err=%v", len(hits), err)
	}
}
