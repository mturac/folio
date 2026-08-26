package ingest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mturac/folio/store"
)

// OCRFunc extracts text from an image path. Tests inject a fake;
// production uses BestOCR (tesseract if installed, else filename).
type OCRFunc func(path string) (string, error)

var imageExt = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true, ".heic": true,
}

// ImportShots walks dir for images and stores OCR text.
func ImportShots(d *store.DB, dir string, ocr OCRFunc) (int, error) {
	if ocr == nil {
		ocr = BestOCR
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if !imageExt[ext] {
			continue
		}
		path := filepath.Join(dir, e.Name())
		text, err := ocr(path)
		if err != nil {
			text = ""
		}
		body := strings.TrimSpace(text)
		if body == "" {
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
	return n, nil
}

// BestOCR uses tesseract when on PATH, otherwise indexes the filename.
func BestOCR(path string) (string, error) {
	if _, err := exec.LookPath("tesseract"); err != nil {
		return strings.ReplaceAll(filepath.Base(path), "_", " "), nil
	}
	out, err := exec.Command("tesseract", path, "stdout", "-l", "eng").Output()
	if err != nil {
		return strings.ReplaceAll(filepath.Base(path), "_", " "), nil
	}
	return string(out), nil
}
