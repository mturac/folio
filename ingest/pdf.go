package ingest

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mturac/folio/store"
)

// ImportPDF extracts text with pdftotext when available, otherwise indexes
// the filename so the file is still findable.
func ImportPDF(d *store.DB, path string) (int, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	if info.IsDir() {
		n := 0
		err := filepath.WalkDir(path, func(p string, e os.DirEntry, walkErr error) error {
			if walkErr != nil || e.IsDir() {
				return walkErr
			}
			if !strings.EqualFold(filepath.Ext(e.Name()), ".pdf") {
				return nil
			}
			c, err := ImportPDF(d, p)
			if err != nil {
				fmt.Fprintf(os.Stderr, "folio: pdf %s: %v\n", p, err)
				return nil
			}
			n += c
			return nil
		})
		return n, err
	}
	if !strings.EqualFold(filepath.Ext(path), ".pdf") {
		return 0, fmt.Errorf("not a pdf: %s", path)
	}
	body, err := pdfText(path)
	if err != nil {
		return 0, err
	}
	if strings.TrimSpace(body) == "" {
		body = filepath.Base(path) + " (no text extracted)"
	}
	_, err = d.Upsert(store.Item{
		Kind:   store.KindPDF,
		Source: path,
		Title:  filepath.Base(path),
		Body:   body,
		When:   info.ModTime(),
	})
	if err != nil {
		return 0, err
	}
	return 1, nil
}

func pdfText(path string) (string, error) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		return strings.ReplaceAll(filepath.Base(path), "_", " "), nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "pdftotext", "-layout", "-nopgbrk", path, "-").Output()
	if err != nil {
		return "", fmt.Errorf("pdftotext failed for %s: %w", path, err)
	}
	return string(out), nil
}
