package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestReplaceChatIndexesMessages(t *testing.T) {
	d, err := Open(filepath.Join(t.TempDir(), "folio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })

	thread := Item{Kind: KindChat, Source: "t.txt", Title: "A, B · 2 msgs", Body: "preview", When: time.Now().UTC()}
	msgs := []Item{
		{Kind: KindChat, Source: MsgSource("t.txt", 1), Title: "A", Body: "hello boarding", When: time.Now().UTC()},
		{Kind: KindChat, Source: MsgSource("t.txt", 2), Title: "B", Body: "ok", When: time.Now().UTC()},
	}
	n, err := d.ReplaceChat(thread, msgs)
	if err != nil || n != 2 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	hits, err := d.Search("boarding")
	if err != nil || len(hits) != 1 {
		t.Fatalf("hits=%d err=%v", len(hits), err)
	}
	if hits[0].Title != "A" || !IsMsgSource(hits[0].Source) {
		t.Fatalf("hit=%+v", hits[0])
	}
	list, err := d.List(KindChat, 10)
	if err != nil || len(list) != 1 {
		t.Fatalf("list threads only: %d %v", len(list), err)
	}
	s, _ := d.Stats()
	if s.ByKind[KindChat] != 1 {
		t.Fatalf("stats=%+v", s)
	}

	// Re-replace shrinks
	n, err = d.ReplaceChat(thread, msgs[:1])
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	total, _ := d.Count()
	if total != 2 {
		t.Fatalf("total=%d", total)
	}
}

func TestSplitMsgSource(t *testing.T) {
	th, seq, ok := SplitMsgSource("path/chat.txt#m000042")
	if !ok || th != "path/chat.txt" || seq != 42 {
		t.Fatalf("%s %d %v", th, seq, ok)
	}
	if IsMsgSource("path/chat.txt") {
		t.Fatal("thread is not msg")
	}
}
