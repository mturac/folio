package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWelcomeEmptyLibrary(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	if err := cmdWelcome(); err != nil {
		t.Fatal(err)
	}
}

func TestInitCreatesInbox(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	if err := cmdInit(); err != nil {
		t.Fatal(err)
	}
	inbox := filepath.Join(dir, ".folio", "inbox")
	if st, err := os.Stat(inbox); err != nil || !st.IsDir() {
		t.Fatalf("inbox missing: %v", err)
	}
}

func TestGuessIngestKind(t *testing.T) {
	cases := map[string]string{
		"photo.PNG":          "shots",
		"WhatsApp Chat.zip":  "chat",
		"messages.html":      "chat",
		"weekly.eml":         "letter",
		"dispatch.html":      "letter",
		"box.mbox":           "letter",
	}
	for name, want := range cases {
		if got := guessIngestKind(name); got != want {
			t.Fatalf("%s: got %s want %s", name, got, want)
		}
	}
}

func TestRunNoArgsDoesNotError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	if err := run(nil); err != nil {
		t.Fatal(err)
	}
}

func TestServeOpenFlagParsed(t *testing.T) {
	// listenAddr should ignore nothing when only --open was stripped by cmdServe;
	// this locks the default bind helper.
	got, err := listenAddr(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "8787") {
		t.Fatalf("got %s", got)
	}
}
