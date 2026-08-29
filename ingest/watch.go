package ingest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mturac/folio/store"
)

// WatchFunc is called when a watched path should be re-ingested.
type WatchFunc func(path string) (int, error)

// WatchPoll polls paths every interval and calls ingest when mtime/size changes.
// It runs until stop is closed. First pass always ingests.
func WatchPoll(paths []string, interval time.Duration, ingest WatchFunc, stop <-chan struct{}, logf func(string, ...any)) error {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	if logf == nil {
		logf = func(string, ...any) {}
	}
	state := map[string]fileSig{}
	tick := time.NewTicker(interval)
	defer tick.Stop()

	scan := func(force bool) {
		for _, root := range paths {
			fi, err := os.Stat(root)
			if err != nil {
				logf("watch: %s: %v", root, err)
				continue
			}
			if !fi.IsDir() {
				info := fi
				sig := fileSig{size: info.Size(), mtime: info.ModTime()}
				prev, seen := state[root]
				if force || !seen || prev != sig {
					state[root] = sig
					n, err := ingest(root)
					if err != nil {
						logf("watch: %s: %v", root, err)
					} else if n > 0 {
						logf("watch: ingested %d from %s", n, root)
					}
				}
				continue
			}
			_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return err
				}
				info, err := d.Info()
				if err != nil {
					return nil
				}
				sig := fileSig{size: info.Size(), mtime: info.ModTime()}
				prev, seen := state[path]
				if !force && seen && prev == sig {
					return nil
				}
				state[path] = sig
				n, err := ingest(path)
				if err != nil {
					logf("watch: %s: %v", path, err)
					return nil
				}
				if n > 0 {
					logf("watch: ingested %d from %s", n, path)
				}
				return nil
			})
		}
	}

	scan(true)
	for {
		select {
		case <-stop:
			return nil
		case <-tick.C:
			scan(false)
		}
	}
}

type fileSig struct {
	size  int64
	mtime time.Time
}

// IngestWatchedPath routes a single file/dir change to the right importer.
func IngestWatchedPath(d *store.DB, kind, path string) (int, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	switch kind {
	case "chat":
		if info.IsDir() {
			return 0, fmt.Errorf("chat watch expects files, got dir %s", path)
		}
		return ImportChatPath(d, path)
	case "letter":
		if info.IsDir() {
			return ImportLetterPath(d, path)
		}
		lower := strings.ToLower(path)
		if !strings.HasSuffix(lower, ".html") && !strings.HasSuffix(lower, ".htm") &&
			!strings.HasSuffix(lower, ".eml") && !strings.HasSuffix(lower, ".mbox") {
			return 0, nil
		}
		return ImportLetterPath(d, path)
	case "shots":
		if info.IsDir() {
			return ImportShots(d, path, BestOCR)
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !imageExt[ext] {
			return 0, nil
		}
		dir := filepath.Dir(path)
		return ImportShots(d, dir, BestOCR)
	case "pdf":
		if info.IsDir() {
			return ImportPDF(d, path)
		}
		if !strings.EqualFold(filepath.Ext(path), ".pdf") {
			return 0, nil
		}
		return ImportPDF(d, path)
	default:
		return 0, fmt.Errorf("unknown watch kind %q", kind)
	}
}
