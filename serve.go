package main

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mturac/folio/store"
)

func cmdServe(argv []string) error {
	addr, err := listenAddr(argv)
	if err != nil {
		return err
	}
	d, err := store.Open(dbPath())
	if err != nil {
		return err
	}
	defer d.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, indexHTML)
	})
	mux.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.RawQuery) > 4096 {
			http.Error(w, "query too long", http.StatusRequestURITooLong)
			return
		}
		q := r.URL.Query().Get("q")
		if len(q) > 256 {
			http.Error(w, "query too long", http.StatusRequestURITooLong)
			return
		}
		var items []store.Item
		var err error
		if q == "" {
			items, err = d.List("", 80)
		} else {
			items, err = d.SearchContext(r.Context(), q)
		}
		if err != nil {
			log.Printf("search: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	})
	mux.HandleFunc("/api/item", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
		if err != nil || id < 1 {
			http.Error(w, "id required", http.StatusBadRequest)
			return
		}
		it, err := d.GetContext(r.Context(), id)
		if err != nil {
			log.Printf("get: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if it == nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<pre>%s</pre>", html.EscapeString(it.Body))
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	fmt.Printf("folio reading room → http://%s  (localhost only)\n", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func listenAddr(argv []string) (string, error) {
	raw := ":8787"
	if len(argv) > 0 {
		raw = argv[0]
	}
	if !strings.Contains(raw, ":") {
		raw = "127.0.0.1:" + raw
	}
	host, port, err := net.SplitHostPort(raw)
	if err != nil {
		return "", fmt.Errorf("invalid listen address %q", argv[0])
	}
	if host == "" {
		host = "127.0.0.1"
	}
	switch host {
	case "127.0.0.1", "localhost", "::1":
	default:
		return "", fmt.Errorf("refusing to bind to non-loopback host %q", host)
	}
	return net.JoinHostPort(host, port), nil
}

const indexHTML = `<!doctype html>
<html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>folio</title>
<style>
  :root { color-scheme: light dark; }
  body { font: 16px/1.5 ui-serif, Georgia, serif; max-width: 720px; margin: 2rem auto; padding: 0 1.2rem; }
  h1 { font-weight: 500; letter-spacing: -0.02em; }
  input { width: 100%; font: inherit; padding: .5rem .7rem; border: 1px solid color-mix(in srgb, currentColor 25%, transparent); background: transparent; border-radius: 6px; }
  .hit { margin: 1.2rem 0; padding-bottom: 1.2rem; border-bottom: 1px solid color-mix(in srgb, currentColor 12%, transparent); }
  .kind { font: 11px/1 ui-sans-serif, system-ui; text-transform: uppercase; letter-spacing: .08em; opacity: .6; }
  .body { opacity: .85; font-size: .95rem; }
  .empty { opacity: .5; margin-top: 2rem; }
</style>
<h1>folio</h1>
<p style="opacity:.7">chats · screenshots · newsletters — on your disk</p>
<input id="q" placeholder="search…" autofocus>
<div id="out"></div>
<script>
const out = document.getElementById('out');
const q = document.getElementById('q');
async function run() {
  const r = await fetch('/api/search?q=' + encodeURIComponent(q.value));
  const items = await r.json();
  if (!items || !items.length) { out.innerHTML = '<p class="empty">nothing here yet — folio ingest …</p>'; return; }
  out.innerHTML = items.map(it =>
    '<div class="hit"><div class="kind">' + (it.Kind||'') + '</div><strong>' +
    escapeHtml(it.Title||'') + '</strong><div class="body">' +
    escapeHtml((it.Body||'').slice(0, 280)) + '</div></div>'
  ).join('');
}
function escapeHtml(s){return s.replace(/[&<>"]/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[c]));}
q.addEventListener('input', () => { clearTimeout(q._t); q._t = setTimeout(run, 120); });
run();
</script>
</html>
`
