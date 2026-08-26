package ingest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mturac/folio/store"
)

// OCRFunc extracts text from an image path. Tests inject a fake;
// production uses BestOCR (tesseract if installed, else filename).
type OCRFunc func(ctx context.Context, path string) (string, error)

var imageExt = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true, ".heic": true,
}

// ImportShots walks dir for images and stores OCR text.
// A failed OCR on one file is reported to stderr and skipped;
// other files still ingest. If every image fails, the joined error is returned.
func ImportShots(d *store.DB, dir string, ocr OCRFunc) (int, error) {
	if ocr == nil {
		ocr = BestOCR
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	n := 0
	var joined error
	ctx := context.Background()
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if !imageExt[ext] {
			continue
		}
		path := filepath.Join(dir, e.Name())
		text, err := ocr(ctx, path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "folio: OCR failed for %s: %v\n", path, err)
			joined = errors.Join(joined, err)
			continue
		}
		body := strings.TrimSpace(text)
		if body == "" {
			// OCR succeeded but found no text — keep the file findable.
			body = e.Name()
		}
		if _, err := d.Add(store.Item{
			Kind:   store.KindShot,
			Source: path,
			Title:  e.Name(),
			Body:   body,
		}); err != nil {
			return n, err
		}
		n++
	}
	if n == 0 && joined != nil {
		return 0, joined
	}
	return n, nil
}

// BestOCR uses tesseract when on PATH. Missing tesseract is not an
// error: we index the filename so the file is still findable. A
// present-but-failing tesseract is an error (no silent filename fake).
func BestOCR(ctx context.Context, path string) (string, error) {
	if _, err := exec.LookPath("tesseract"); err != nil {
		return strings.ReplaceAll(filepath.Base(path), "_", " "), nil
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "tesseract", path, "stdout", "-l", "eng").Output()
	if err != nil {
		return "", fmt.Errorf("tesseract failed for %s: %w", path, err)
	}
	return string(out), nil
}
