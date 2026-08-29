package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/mturac/folio/store"
)

func TestListenAddrLoopbackOnly(t *testing.T) {
	got, err := listenAddr(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "127.0.0.1:8787" {
		t.Fatalf("default: %s", got)
	}
	got, err = listenAddr([]string{"8790"})
	if err != nil || got != "127.0.0.1:8790" {
		t.Fatalf("port: %s %v", got, err)
	}
	if _, err := listenAddr([]string{"0.0.0.0:8787"}); err == nil {
		t.Fatal("must refuse 0.0.0.0")
	}
}

func TestAPISearchAndItem(t *testing.T) {
	dir := t.TempDir()
	dbp := filepath.Join(dir, "folio.db")
	d, err := store.Open(dbp)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	id, err := d.Upsert(store.Item{
		Kind: store.KindChat, Source: "c", Title: "Family", Body: "boarding pass Friday",
		When: time.Date(2023, 12, 31, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		s, _ := d.Stats()
		json.NewEncoder(w).Encode(s)
	})
	mux.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		kind := r.URL.Query().Get("kind")
		var items []store.Item
		if q == "" {
			items, _ = d.ListContext(r.Context(), kind, 80)
		} else {
			items, _ = d.SearchContext(r.Context(), q, kind)
		}
		json.NewEncoder(w).Encode(items)
	})
	mux.HandleFunc("/api/item", func(w http.ResponseWriter, r *http.Request) {
		it, _ := d.GetContext(r.Context(), id)
		json.NewEncoder(w).Encode(it)
	})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/search?q=boarding&kind=chat", nil))
	if rec.Code != 200 {
		t.Fatalf("status %d", rec.Code)
	}
	var hits []store.Item
	if err := json.NewDecoder(rec.Body).Decode(&hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("hits=%d", len(hits))
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/item?id=1", nil))
	var it store.Item
	json.NewDecoder(rec.Body).Decode(&it)
	if it.Title != "Family" {
		t.Fatalf("item=%+v", it)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/stats", nil))
	var s store.Stats
	json.NewDecoder(rec.Body).Decode(&s)
	if s.Total != 1 {
		t.Fatalf("stats=%+v", s)
	}
}

func TestMediaServesShotOnly(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "wifi.png")
	if err := os.WriteFile(img, []byte("fakepng"), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := store.Open(filepath.Join(dir, "folio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	id, err := d.Upsert(store.Item{Kind: store.KindShot, Source: img, Title: "wifi.png", Body: "wifi"})
	if err != nil {
		t.Fatal(err)
	}
	chatID, err := d.Upsert(store.Item{Kind: store.KindChat, Source: "c", Title: "c", Body: "x"})
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/media", func(w http.ResponseWriter, r *http.Request) {
		rid, _ := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
		it, _ := d.GetContext(r.Context(), rid)
		if it == nil || it.Kind != store.KindShot {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, it.Source)
	})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/media?id="+strconv.FormatInt(id, 10), nil))
	if rec.Code != 200 || string(rec.Body.Bytes()) != "fakepng" {
		t.Fatalf("shot media: %d %q", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/media?id="+strconv.FormatInt(chatID, 10), nil))
	if rec.Code != 404 {
		t.Fatalf("chat must 404, got %d", rec.Code)
	}
}
