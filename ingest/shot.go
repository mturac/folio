package ingest

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/mturac/folio/store"
)

// OCRFunc extracts text from an image path. Tests inject a fake;
// production uses BestOCR (tesseract if installed, else filename).
type OCRFunc func(ctx context.Context, path string) (string, error)

var imageExt = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true, ".heic": true,
}

type shotJob struct {
	path  string
	title string
	when  time.Time
}

type shotResult struct {
	job  shotJob
	body string
	err  error
}

// ImportShots walks dir recursively for images and stores OCR text.
// OCR runs with a small worker pool; SQLite upserts stay serial.
// A failed OCR on one file is reported to stderr and skipped;
// other files still ingest. If every image fails, the joined error is returned.
func ImportShots(d *store.DB, dir string, ocr OCRFunc) (int, error) {
	if ocr == nil {
		ocr = BestOCR
	}
	var jobs []shotJob
	err := filepath.WalkDir(dir, func(path string, e fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if e.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if !imageExt[ext] {
			return nil
		}
		j := shotJob{path: path, title: e.Name()}
		if info, err := e.Info(); err == nil {
			j.when = info.ModTime()
		}
		jobs = append(jobs, j)
		return nil
	})
	if err != nil {
		return 0, err
	}
	if len(jobs) == 0 {
		return 0, nil
	}

	workers := runtime.NumCPU()
	if workers > 4 {
		workers = 4
	}
	if workers > len(jobs) {
		workers = len(jobs)
	}

	ctx := context.Background()
	in := make(chan shotJob)
	out := make(chan shotResult, len(jobs))
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range in {
				text, err := ocr(ctx, j.path)
				body := strings.TrimSpace(text)
				if err == nil && body == "" {
					body = j.title + " (no text recognized)"
				}
				out <- shotResult{job: j, body: body, err: err}
			}
		}()
	}
	go func() {
		for _, j := range jobs {
			in <- j
		}
		close(in)
		wg.Wait()
		close(out)
	}()

	n := 0
	var joined error
	for res := range out {
		if res.err != nil {
			fmt.Fprintf(os.Stderr, "folio: OCR failed for %s: %v\n", res.job.path, res.err)
			joined = errors.Join(joined, res.err)
			continue
		}
		if _, err := d.Upsert(store.Item{
			Kind:   store.KindShot,
			Source: res.job.path,
			Title:  res.job.title,
			Body:   res.body,
			When:   res.job.when,
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
	// eng+tur covers the common Turkish+English screenshot case when
	// both traineddata packs are installed; fall back to eng only.
	out, err := exec.CommandContext(ctx, "tesseract", path, "stdout", "-l", "eng+tur").Output()
	if err != nil {
		out, err = exec.CommandContext(ctx, "tesseract", path, "stdout", "-l", "eng").Output()
		if err != nil {
			return "", fmt.Errorf("tesseract failed for %s: %w", path, err)
		}
	}
	return string(out), nil
}
