package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/mturac/folio/store"
)

func cmdDoctor() error {
	fmt.Printf("folio %s\n", version)
	fmt.Printf("go:    %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)

	db := dbPath()
	fmt.Printf("db:    %s\n", db)
	if fi, err := os.Stat(db); err != nil {
		fmt.Println("       (missing — will be created on first ingest)")
	} else {
		fmt.Printf("       %d bytes\n", fi.Size())
		d, err := store.Open(db)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}
		defer d.Close()
		s, err := d.Stats()
		if err != nil {
			return err
		}
		fmt.Printf("items: %d", s.Total)
		for _, k := range []string{store.KindChat, store.KindShot, store.KindLetter} {
			if n := s.ByKind[k]; n > 0 {
				fmt.Printf("  %s=%d", k, n)
			}
		}
		fmt.Println()
	}

	if p, err := exec.LookPath("tesseract"); err != nil {
		fmt.Println("ocr:   tesseract not found — filenames still indexed")
	} else {
		fmt.Printf("ocr:   %s\n", p)
		out, err := exec.Command("tesseract", "--list-langs").CombinedOutput()
		if err == nil {
			langs := string(out)
			hasEng := containsWord(langs, "eng")
			hasTur := containsWord(langs, "tur")
			fmt.Printf("langs: eng=%v tur=%v\n", hasEng, hasTur)
		}
	}

	home, _ := os.UserHomeDir()
	fmt.Printf("home:  %s\n", filepath.Join(home, ".folio"))
	fmt.Println("ok")
	return nil
}

func containsWord(s, w string) bool {
	for _, line := range splitLines(s) {
		if line == w {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			out = append(out, line)
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}
