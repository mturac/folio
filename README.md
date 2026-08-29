# folio

<p align="center">
  <strong>English</strong> · <a href="README.tr.md">Türkçe</a>
</p>

<p align="center">
  <a href="https://github.com/mturac/folio/releases/latest"><img alt="Release" src="https://img.shields.io/github/v/release/mturac/folio?style=flat-square&color=0f464e" /></a>
  <a href="LICENSE"><img alt="License" src="https://img.shields.io/github/license/mturac/folio?style=flat-square&color=0f464e" /></a>
  <a href="https://github.com/mturac/folio/actions/workflows/ci.yml"><img alt="CI" src="https://img.shields.io/github/actions/workflow/status/mturac/folio/ci.yml?branch=main&style=flat-square&label=ci" /></a>
  <img alt="Go" src="https://img.shields.io/badge/go-1.25-00ADD8?style=flat-square" />
</p>

<p align="center">
  <strong>Chats. Screenshots. Newsletters. Searchable. On your disk.</strong><br />
  No account. No cloud. One binary.
</p>

<p align="center">
  <img src="docs/assets/reading-room.png" alt="folio reading room — local search across chats, screenshots, and newsletters" width="920" />
</p>

---

## Why folio

Your boarding pass is in a screenshot. The wifi code is in a WhatsApp export.
Last week’s newsletter is an `.eml` in Downloads. Folio indexes them locally
so you can find them again — in the terminal or in a small reading room that
never leaves your machine.

| You keep | Folio does |
| --- | --- |
| Files on disk | Full-text search (SQLite FTS) |
| Messenger exports | Message-level chat index |
| Screenshots | OCR when Tesseract is installed |
| Privacy | Localhost-only UI · no telemetry |

---

## 60 seconds

```bash
curl -fsSL https://raw.githubusercontent.com/mturac/folio/main/install.sh | bash
folio init
folio ingest chat ~/Downloads/WhatsApp\ Chat.zip
folio ingest shots ~/Desktop/Screenshots
folio serve --open
```

Search in the browser, or drop a file onto the page.

<p align="center">
  <img src="docs/assets/reading-room-search.png" alt="Searching for boarding across chats and screenshots" width="920" />
</p>

Prebuilt binaries for macOS, Linux, and Windows ship on
[Releases](https://github.com/mturac/folio/releases). The install script
prefers a release archive; otherwise it falls back to `go install`.

---

## Reading room

`folio serve --open` starts a localhost UI:

- Search with highlighted matches
- Filter by chat · shots · letters · pdf
- Open an item for the full body or image
- Drag-and-drop to ingest without leaving the page

<p align="center">
  <img src="docs/assets/reading-room-detail.png" alt="Opening a screenshot hit in the reading room" width="920" />
</p>

Keyboard: `j` / `k` move · `Enter` opens · `Esc` closes.

---

## Install

**Script (macOS / Linux)**

```bash
curl -fsSL https://raw.githubusercontent.com/mturac/folio/main/install.sh | bash
```

**Go**

```bash
go install github.com/mturac/folio@latest
```

**From source**

```bash
git clone https://github.com/mturac/folio
cd folio && make install
```

### Optional tools

| Tool | Why |
| --- | --- |
| [Tesseract](https://tesseract-ocr.github.io/) | OCR text inside screenshots |
| `pdftotext` (poppler) | Extract text from PDFs |

Without them, filenames (and whatever text the format already carries) are
still indexed.

```bash
# macOS
brew install tesseract tesseract-lang poppler
# Debian / Ubuntu
sudo apt install tesseract-ocr tesseract-ocr-eng tesseract-ocr-tur poppler-utils
```

---

## Commands

| Do this | Type this |
| --- | --- |
| First-time setup | `folio init` |
| Add WhatsApp / Telegram / Signal | `folio ingest chat <file>` |
| Add screenshots | `folio ingest shots <folder>` |
| Add newsletters | `folio ingest letter <file-or-folder>` |
| Add PDFs | `folio ingest pdf <file-or-folder>` |
| Open the reading room | `folio serve --open` |
| Search in the terminal | `folio search "boarding pass"` |
| Library counts | `folio stats` |
| Watch a path | `folio watch shots ~/Desktop/Screenshots` |
| Check the install | `folio doctor` |
| Export the library | `folio export json` |

`folio` with no arguments prints the next step for an empty or full library.

---

## What it accepts

| Kind | Files |
| --- | --- |
| **Chat** | WhatsApp `_chat.txt` / zip · Telegram HTML / text · Signal markdown / JSONL |
| **Shots** | png / jpg / webp / … (recursive; OCR when Tesseract is installed) |
| **Letter** | `.html` / `.eml` / `.mbox`, or a folder of them |
| **PDF** | `.pdf` (text via `pdftotext` when available; otherwise filename) |

Data lives under `~/.folio/` (SQLite + inbox). Nothing is uploaded.

---

## Privacy

- No accounts, no sync service, no telemetry
- `folio serve` binds **localhost only**
- Folio only sees files you ingest or drop into the reading room

See [SECURITY.md](SECURITY.md).

---

## Status

**v0.6** · heading to **v0.57** public launch · [roadmap](ROADMAP.md) · MIT

Contributing: [CONTRIBUTING.md](CONTRIBUTING.md)

---

<p align="center">
  <sub>Türkçe sürüm: <a href="README.tr.md">README.tr.md</a></sub>
</p>
