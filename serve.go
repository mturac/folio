package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mturac/folio/ingest"
	"github.com/mturac/folio/store"
)

func cmdServe(argv []string) error {
	open := false
	var addrArgs []string
	for _, a := range argv {
		switch a {
		case "--open", "-o":
			open = true
		default:
			addrArgs = append(addrArgs, a)
		}
	}
	addr, err := listenAddr(addrArgs)
	if err != nil {
		return err
	}
	d, err := store.Open(dbPath())
	if err != nil {
		return err
	}
	defer d.Close()
	inbox := filepath.Join(folioDir(), "inbox")
	os.MkdirAll(inbox, 0o700)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, indexHTML)
	})
	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		s, err := d.Stats()
		if err != nil {
			log.Printf("stats: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s)
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
		kind := r.URL.Query().Get("kind")
		switch kind {
		case "", store.KindChat, store.KindShot, store.KindLetter:
		default:
			http.Error(w, "invalid kind", http.StatusBadRequest)
			return
		}
		var items []store.Item
		var err error
		if q == "" {
			items, err = d.ListContext(r.Context(), kind, 80)
		} else {
			items, err = d.SearchContext(r.Context(), q, kind)
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
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(it)
	})
	mux.HandleFunc("/api/media", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
		if err != nil || id < 1 {
			http.Error(w, "id required", http.StatusBadRequest)
			return
		}
		it, err := d.GetContext(r.Context(), id)
		if err != nil {
			log.Printf("media: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if it == nil || it.Kind != store.KindShot {
			http.NotFound(w, r)
			return
		}
		path := it.Source
		if !filepath.IsAbs(path) {
			http.Error(w, "refusing relative media path", http.StatusForbidden)
			return
		}
		fi, err := os.Stat(path)
		if err != nil || fi.IsDir() {
			http.NotFound(w, r)
			return
		}
		if fi.Size() > 40<<20 {
			http.Error(w, "media too large", http.StatusRequestEntityTooLarge)
			return
		}
		ctype := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
		if ctype == "" {
			ctype = "application/octet-stream"
		}
		w.Header().Set("Content-Type", ctype)
		w.Header().Set("Cache-Control", "private, max-age=60")
		http.ServeFile(w, r, path)
	})
	mux.HandleFunc("/api/ingest", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 64<<20)
		if err := r.ParseMultipartForm(64 << 20); err != nil {
			http.Error(w, "bad multipart", http.StatusBadRequest)
			return
		}
		file, hdr, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "file required", http.StatusBadRequest)
			return
		}
		defer file.Close()
		name := filepath.Base(hdr.Filename)
		name = ingest.SanitizeSource(name)
		if name == "" || name == "." || name == ".." {
			http.Error(w, "bad filename", http.StatusBadRequest)
			return
		}
		dst := filepath.Join(inbox, fmt.Sprintf("%d_%s", time.Now().UnixNano(), name))
		out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if _, err := io.Copy(out, file); err != nil {
			out.Close()
			os.Remove(dst)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		out.Close()

		kind := r.FormValue("kind")
		if kind == "" {
			kind = guessIngestKind(name)
		}
		n, err := ingestDropped(d, kind, dst)
		if err != nil {
			log.Printf("ingest drop: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ingested": n, "kind": kind})
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	url := "http://" + addr
	fmt.Printf("folio reading room → %s  (localhost only)\n", url)
	if open {
		go func() {
			time.Sleep(250 * time.Millisecond)
			openBrowser(url)
		}()
	}
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func guessIngestKind(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".png"), strings.HasSuffix(lower, ".jpg"),
		strings.HasSuffix(lower, ".jpeg"), strings.HasSuffix(lower, ".webp"),
		strings.HasSuffix(lower, ".gif"), strings.HasSuffix(lower, ".heic"):
		return "shots"
	case strings.HasSuffix(lower, ".eml"), strings.HasSuffix(lower, ".mbox"):
		return "letter"
	case strings.HasSuffix(lower, ".zip"), strings.HasSuffix(lower, ".txt"):
		return "chat"
	case strings.HasSuffix(lower, ".html"), strings.HasSuffix(lower, ".htm"):
		if strings.Contains(lower, "telegram") || strings.Contains(lower, "messages") ||
			strings.Contains(lower, "chat") || strings.Contains(lower, "whatsapp") {
			return "chat"
		}
		return "letter"
	default:
		return "letter"
	}
}

func ingestDropped(d *store.DB, kind, path string) (int, error) {
	switch kind {
	case "chat":
		return ingest.ImportChatPath(d, path)
	case "letter":
		return ingest.ImportLetterPath(d, path)
	case "shots", "shot":
		return ingest.ImportShots(d, filepath.Dir(path), ingest.BestOCR)
	default:
		return 0, fmt.Errorf("unknown kind %q", kind)
	}
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
  :root {
    color-scheme: light dark;
    --ink: #14212b;
    --line: color-mix(in srgb, currentColor 14%, transparent);
    --paper: #e7eef2;
    --accent: #0f6e6e;
    --mark: color-mix(in srgb, #c9a227 45%, transparent);
  }
  @media (prefers-color-scheme: dark) {
    :root { --paper: #0f1418; --ink: #e6eef2; --accent: #5ec4c0; }
  }
  * { box-sizing: border-box; }
  body {
    margin: 0; min-height: 100vh;
    font: 16px/1.55 "Iowan Old Style", "Palatino Linotype", Palatino, Georgia, serif;
    color: var(--ink); background:
      radial-gradient(1100px 520px at 8% -8%, color-mix(in srgb, var(--accent) 14%, transparent), transparent 58%),
      radial-gradient(800px 420px at 100% 0%, color-mix(in srgb, currentColor 5%, transparent), transparent 50%),
      var(--paper);
  }
  .wrap { max-width: 740px; margin: 0 auto; padding: 2.2rem 1.25rem 4rem; }
  header { margin-bottom: 1.4rem; }
  h1 { font-weight: 500; letter-spacing: -0.03em; font-size: 2.1rem; margin: 0 0 .25rem; }
  .tag { opacity: .65; margin: 0; font-size: .95rem; }
  .stats { font: 12px/1.4 ui-sans-serif, system-ui, sans-serif; opacity: .55; margin-top: .6rem; letter-spacing: .02em; }
  .toolbar { display: flex; flex-direction: column; gap: .75rem; margin: 1.4rem 0 1.6rem; }
  input {
    width: 100%; font: inherit; padding: .65rem .85rem;
    border: 1px solid var(--line); background: color-mix(in srgb, var(--paper) 70%, white 30%);
    border-radius: 8px; color: inherit; outline: none;
  }
  input:focus { border-color: color-mix(in srgb, var(--accent) 55%, var(--line)); }
  .filters { display: flex; flex-wrap: wrap; gap: .4rem; }
  .filters button {
    font: 12px/1 ui-sans-serif, system-ui, sans-serif; letter-spacing: .06em; text-transform: uppercase;
    padding: .4rem .7rem; border: 1px solid var(--line); background: transparent; color: inherit;
    border-radius: 999px; cursor: pointer;
  }
  .filters button[aria-pressed="true"] {
    background: var(--accent);
    color: #f4fbfb;
    border-color: var(--accent);
  }
  @media (prefers-color-scheme: dark) {
    .filters button[aria-pressed="true"] { color: #0b1518; }
  }
  .hit {
    display: block; width: 100%; text-align: left; cursor: pointer;
    margin: 0; padding: 1.05rem 0; border: 0; border-bottom: 1px solid var(--line);
    background: transparent; color: inherit; font: inherit;
  }
  .hit:focus { outline: 2px solid color-mix(in srgb, var(--accent) 55%, transparent); outline-offset: 2px; }
  .hit:hover .title { color: var(--accent); }
  .meta { display: flex; gap: .65rem; align-items: baseline; margin-bottom: .2rem; }
  .kind { font: 11px/1 ui-sans-serif, system-ui, sans-serif; text-transform: uppercase; letter-spacing: .08em; opacity: .55; }
  .when { font: 11px/1 ui-sans-serif, system-ui, sans-serif; opacity: .45; }
  .title { font-weight: 600; }
  .body { opacity: .8; font-size: .95rem; margin-top: .25rem; }
  .body mark, .sheet mark { background: var(--mark); color: inherit; padding: 0 .1em; }
  .empty { opacity: .5; margin-top: 2rem; }
  .detail {
    position: fixed; inset: 0; background: color-mix(in srgb, #000 45%, transparent);
    display: none; align-items: flex-end; justify-content: center; padding: 1rem;
    z-index: 20;
  }
  .detail.open { display: flex; }
  .sheet {
    width: min(720px, 100%); max-height: min(88vh, 900px); overflow: auto;
    background: var(--paper); color: var(--ink); border-radius: 14px 14px 10px 10px;
    padding: 1.25rem 1.25rem 1.6rem; box-shadow: 0 20px 60px color-mix(in srgb, #000 35%, transparent);
    animation: rise .22s ease-out;
  }
  @keyframes rise { from { transform: translateY(18px); opacity: .5; } to { transform: none; opacity: 1; } }
  .sheet header { display: flex; justify-content: space-between; gap: 1rem; align-items: start; }
  .sheet pre {
    white-space: pre-wrap; word-break: break-word; font: inherit; margin: 1rem 0 0;
    opacity: .9; line-height: 1.6;
  }
  .sheet img.media {
    display: block; width: 100%; max-height: 42vh; object-fit: contain; margin-top: 1rem;
    background: color-mix(in srgb, currentColor 6%, transparent); border-radius: 8px;
  }
  .close {
    font: 12px/1 ui-sans-serif, system-ui, sans-serif; letter-spacing: .06em; text-transform: uppercase;
    border: 1px solid var(--line); background: transparent; color: inherit; border-radius: 999px;
    padding: .4rem .7rem; cursor: pointer;
  }
  .src { font: 12px/1.4 ui-sans-serif, system-ui, sans-serif; opacity: .45; margin-top: .35rem; word-break: break-all; }
  .drop {
    position: fixed; inset: 0; display: none; align-items: center; justify-content: center;
    background: color-mix(in srgb, var(--accent) 18%, transparent); z-index: 40;
    font: 1.2rem/1.4 "Iowan Old Style", Palatino, Georgia, serif; pointer-events: none;
  }
  .drop.on { display: flex; }
  .drop span {
    padding: 1rem 1.4rem; border: 1px dashed var(--accent); border-radius: 12px;
    background: var(--paper);
  }
  .hint { font: 12px/1.4 ui-sans-serif, system-ui, sans-serif; opacity: .5; margin-top: .35rem; }
</style>
<div class="wrap">
  <header>
    <h1>folio</h1>
    <p class="tag">chats · screenshots · newsletters — on your disk</p>
    <div class="stats" id="stats"></div>
    <p class="hint">drop a chat export, screenshot, or .eml anywhere on this page</p>
  </header>
  <div class="toolbar">
    <input id="q" placeholder="search boarding pass, wifi, weekly…" autofocus>
    <div class="filters" role="group" aria-label="kind">
      <button type="button" data-kind="" aria-pressed="true">all</button>
      <button type="button" data-kind="chat" aria-pressed="false">chat</button>
      <button type="button" data-kind="shot" aria-pressed="false">shots</button>
      <button type="button" data-kind="letter" aria-pressed="false">letters</button>
    </div>
  </div>
  <div id="out"></div>
</div>
<div class="drop" id="drop"><span>drop to add to folio</span></div>
<div class="detail" id="detail" role="dialog" aria-modal="true">
  <div class="sheet">
    <header>
      <div>
        <div class="meta"><span class="kind" id="dKind"></span><span class="when" id="dWhen"></span></div>
        <div class="title" id="dTitle"></div>
        <div class="src" id="dSrc"></div>
      </div>
      <button class="close" type="button" id="close">close</button>
    </header>
    <img class="media" id="dImg" alt="" hidden>
    <pre id="dBody"></pre>
  </div>
</div>
<script>
const out = document.getElementById('out');
const q = document.getElementById('q');
const statsEl = document.getElementById('stats');
const detail = document.getElementById('detail');
const dImg = document.getElementById('dImg');
let kind = '';
let items = [];

function escapeHtml(s){
  return String(s||'').replace(/[&<>"]/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[c]));
}
function highlight(s){
  const raw = escapeHtml(s);
  const terms = (q.value||'').trim().split(/\s+/).filter(Boolean).slice(0, 6);
  if (!terms.length) return raw;
  let out = raw;
  for (const t of terms) {
    const re = new RegExp('(' + t.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + ')', 'ig');
    out = out.replace(re, '<mark>$1</mark>');
  }
  return out;
}
function fmtWhen(iso){
  if (!iso || iso.startsWith('0001')) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  return d.toLocaleDateString(undefined, {year:'numeric', month:'short', day:'numeric'});
}
async function loadStats(){
  try {
    const s = await (await fetch('/api/stats')).json();
    const parts = [];
    if (s.byKind?.chat) parts.push(s.byKind.chat + ' chats');
    if (s.byKind?.shot) parts.push(s.byKind.shot + ' shots');
    if (s.byKind?.letter) parts.push(s.byKind.letter + ' letters');
    statsEl.textContent = s.total ? (s.total + ' items · ' + parts.join(' · ')) : 'empty library — folio ingest …';
  } catch (_) { statsEl.textContent = ''; }
}
async function run() {
  const url = '/api/search?q=' + encodeURIComponent(q.value) + (kind ? '&kind=' + encodeURIComponent(kind) : '');
  const r = await fetch(url);
  items = await r.json();
  if (!items || !items.length) {
    out.innerHTML = '<p class="empty">nothing here yet — folio ingest chat|shots|letter …</p>';
    return;
  }
  out.innerHTML = items.map((it, i) =>
    '<button class="hit" type="button" data-i="'+i+'"><div class="meta"><span class="kind">'+
    escapeHtml(it.Kind||'')+'</span><span class="when">'+escapeHtml(fmtWhen(it.When))+
    '</span></div><div class="title">'+highlight(it.Title||'')+
    '</div><div class="body">'+highlight((it.Body||'').slice(0, 280))+'</div></button>'
  ).join('');
}
function openItem(it){
  document.getElementById('dKind').textContent = it.Kind || '';
  document.getElementById('dWhen').textContent = fmtWhen(it.When);
  document.getElementById('dTitle').textContent = it.Title || '';
  document.getElementById('dSrc').textContent = it.Source || '';
  document.getElementById('dBody').innerHTML = highlight(it.Body || '');
  if (it.Kind === 'shot' && it.ID) {
    dImg.hidden = false;
    dImg.src = '/api/media?id=' + it.ID;
  } else {
    dImg.hidden = true;
    dImg.removeAttribute('src');
  }
  detail.classList.add('open');
}
out.addEventListener('click', async (e) => {
  const btn = e.target.closest('.hit');
  if (!btn) return;
  const it = items[+btn.dataset.i];
  if (!it) return;
  try {
    const full = await (await fetch('/api/item?id=' + it.ID)).json();
    openItem(full);
  } catch (_) { openItem(it); }
});
document.getElementById('close').onclick = () => detail.classList.remove('open');
detail.addEventListener('click', (e) => { if (e.target === detail) detail.classList.remove('open'); });
document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape') detail.classList.remove('open');
  if (detail.classList.contains('open')) return;
  if (e.target === q) return;
  if (e.key === 'j' || e.key === 'k') {
    const hits = [...out.querySelectorAll('.hit')];
    if (!hits.length) return;
    const cur = out.querySelector('.hit:focus');
    let i = cur ? hits.indexOf(cur) : -1;
    i = e.key === 'j' ? Math.min(hits.length - 1, i + 1) : Math.max(0, i <= 0 ? 0 : i - 1);
    hits[i].focus();
    e.preventDefault();
  }
  if (e.key === '/' && e.target !== q) { q.focus(); e.preventDefault(); }
});
document.querySelectorAll('.filters button').forEach(b => {
  b.addEventListener('click', () => {
    kind = b.dataset.kind || '';
    document.querySelectorAll('.filters button').forEach(x => x.setAttribute('aria-pressed', String(x === b)));
    run();
  });
});
q.addEventListener('input', () => { clearTimeout(q._t); q._t = setTimeout(run, 120); });
const drop = document.getElementById('drop');
let dragDepth = 0;
window.addEventListener('dragenter', (e) => { e.preventDefault(); dragDepth++; drop.classList.add('on'); });
window.addEventListener('dragleave', () => { dragDepth = Math.max(0, dragDepth - 1); if (!dragDepth) drop.classList.remove('on'); });
window.addEventListener('dragover', (e) => e.preventDefault());
window.addEventListener('drop', async (e) => {
  e.preventDefault(); dragDepth = 0; drop.classList.remove('on');
  const files = [...(e.dataTransfer?.files || [])];
  if (!files.length) return;
  for (const f of files) {
    const fd = new FormData();
    fd.append('file', f, f.name);
    try {
      const r = await fetch('/api/ingest', { method: 'POST', body: fd });
      if (!r.ok) { console.warn(await r.text()); continue; }
    } catch (err) { console.warn(err); }
  }
  loadStats(); run();
});
loadStats();
run();
</script>
</html>
`
