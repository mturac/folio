package ingest

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mturac/folio/store"
)

func TestScreenshotOCRIsSearchable(t *testing.T) {
	dir := t.TempDir()
	d, err := store.Open(filepath.Join(dir, "folio.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { d.Close() })

	shot := filepath.Join(dir, "wifi.png")
	if err := os.WriteFile(shot, []byte("fake-png"), 0o644); err != nil {
		t.Fatal(err)
	}

	ocr := func(ctx context.Context, path string) (string, error) {
		if path != shot {
			t.Fatalf("ocr called with %s", path)
		}
		return "Wi-Fi password: hunter2 network: cafe-guest", nil
	}

	n, err := ImportShots(d, dir, ocr)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if n != 1 {
		t.Fatalf("want 1 shot, got %d", n)
	}

	hits, err := d.Search("hunter2")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 1 || hits[0].Kind != store.KindShot {
		t.Fatalf("ocr text must be searchable: %+v", hits)
	}
	if hits[0].When.IsZero() {
		t.Fatal("shot when should come from mtime")
	}
}

func TestScreenshotImportSkipsNonImages(t *testing.T) {
	dir := t.TempDir()
	d, err := store.Open(filepath.Join(dir, "folio.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { d.Close() })

	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("not an image"), 0o644)
	n, err := ImportShots(d, dir, func(context.Context, string) (string, error) {
		t.Fatal("ocr must not run on non-images")
		return "", nil
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if n != 0 {
		t.Fatalf("want 0, got %d", n)
	}
}

func TestScreenshotImportWalksSubdirs(t *testing.T) {
	dir := t.TempDir()
	d, err := store.Open(filepath.Join(dir, "folio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })

	nested := filepath.Join(dir, "2024", "trip")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	shot := filepath.Join(nested, "gate.png")
	if err := os.WriteFile(shot, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	n, err := ImportShots(d, dir, func(_ context.Context, path string) (string, error) {
		return "gate B12", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("want recursive 1, got %d", n)
	}
}
