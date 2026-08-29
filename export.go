package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mturac/folio/store"
)

func cmdExport(argv []string) error {
	format := "json"
	if len(argv) > 0 {
		format = strings.ToLower(argv[0])
	}
	d, err := store.Open(dbPath())
	if err != nil {
		return err
	}
	defer d.Close()
	items, err := d.List("", 100000)
	if err != nil {
		return err
	}
	switch format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(items)
	case "md", "markdown":
		for _, it := range items {
			when := ""
			if !it.When.IsZero() {
				when = it.When.UTC().Format(time.RFC3339)
			}
			fmt.Printf("## [%s] %s\n\n", it.Kind, it.Title)
			if when != "" {
				fmt.Printf("- when: `%s`\n", when)
			}
			fmt.Printf("- source: `%s`\n\n%s\n\n---\n\n", it.Source, it.Body)
		}
		return nil
	default:
		return fmt.Errorf("export format must be json or md, got %q", format)
	}
}
