package main

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"strings"

	"github.com/mturac/folio/store"
)

func cmdServe(argv []string) {
	addr := ":8787"
	if len(argv) > 0 {
		addr = argv[0]
		if !strings.Contains(addr, ":") {
			addr = ":" + addr
		}
	}
	d := openDB()
	defer d.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, indexHTML)
	})
	mux.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		var items []store.Item
		var err error
		if q == "" {
			items, err = d.List("", 80)
		} else {
			items, err = d.Search(q)
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	})
	mux.HandleFunc("/api/item", func(w http.ResponseWriter, r *http.Request) {
		// list already returns body; this is a cheap HTML view helper
		q := r.URL.Query().Get("q")
		items, err := d.Search(q)
		if err != nil || len(items) == 0 {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<pre>%s</pre>", html.EscapeString(items[0].Body))
	})

	fmt.Printf("folio reading room → http://127.0.0.1%s  (localhost only)\n", addr)
	if err := http.ListenAndServe("127.0.0.1"+addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "folio: %v\n", err)
		os.Exit(1)
	}
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
