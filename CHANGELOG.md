# Changelog

## 0.5.0

- GitHub Actions + GoReleaser publish linux/darwin/windows binaries on `v*` tags
- install.sh prefers release archives; README points at Releases

## 0.4.0

- Message-level chat index: each WhatsApp/Telegram message is its own FTS row
- Thread summary kept for lists/stats; re-ingest replaces `#m*` children cleanly
- Reading room labels message hits as `chat · msg`

## 0.3.0

- First-run welcome (`folio` with no args), `folio init`, `folio help`
- `folio serve --open` launches the browser
- Drag-and-drop ingest in the reading room (`POST /api/ingest`)
- `install.sh` for curl|bash setup
- Goreleaser stub; roadmap train to **v0.57**

## 0.2.0

- Upsert re-ingest, delete, stats; `occurred_at` on items
- FTS update/delete triggers; CLI search snippets
- Chat timestamps + message counts; Telegram HTML/text ingest
- Recursive screenshot walk with parallel OCR workers; multipart EML
- Letter folders + mbox split into per-message items
- Reading room: kind filters, detail sheet, shot images, counts, highlights, j/k
- `folio watch`, `version`, `rm`, `doctor`, `export`
- CI, Makefile, CONTRIBUTING, SECURITY, issue templates, ROADMAP

## 0.1.0

- Initial WhatsApp / shots / letter ingest, FTS search, localhost serve
