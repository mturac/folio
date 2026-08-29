# folio

> Chats. Screenshots. Newsletters. Searchable. On your disk.

No account. No cloud. One binary.

## 60 seconds

```bash
curl -fsSL https://raw.githubusercontent.com/mturac/folio/main/install.sh | bash
folio init
folio ingest chat ~/Downloads/WhatsApp\ Chat.zip
folio ingest shots ~/Desktop/Screenshots
folio serve --open
```

Then search in the browser, or drop a file onto the page.

Prebuilt binaries (macOS / Linux / Windows) ship on
[Releases](https://github.com/mturac/folio/releases). The install script
prefers a release archive when one exists, otherwise uses `go install`.

## Install

**Script (macOS / Linux):**

```bash
curl -fsSL https://raw.githubusercontent.com/mturac/folio/main/install.sh | bash
```

**Go:**

```bash
go install github.com/mturac/folio@latest
```

**From source:**

```bash
git clone https://github.com/mturac/folio
cd folio && make install
```

Optional OCR for screenshots — [Tesseract](https://tesseract-ocr.github.io/).
Without it, filenames are still indexed.

```bash
# macOS
brew install tesseract tesseract-lang
# Debian/Ubuntu
sudo apt install tesseract-ocr tesseract-ocr-eng tesseract-ocr-tur
```

## Commands

| Do this | Type this |
|---------|-----------|
| First-time setup | `folio init` |
| Add WhatsApp / Telegram | `folio ingest chat <file>` |
| Add screenshots | `folio ingest shots <folder>` |
| Add newsletters | `folio ingest letter <file-or-folder>` |
| Add PDFs | `folio ingest pdf <file-or-folder>` |
| Open the reading room | `folio serve --open` |
| Search in the terminal | `folio search "boarding pass"` |
| Check the install | `folio doctor` |

`folio` with no arguments prints the next step for an empty or full library.

## What it accepts

| Kind | Files |
|------|-------|
| Chat | WhatsApp `_chat.txt` / zip, Telegram HTML/text, Signal markdown / JSONL |
| Shots | png / jpg / webp / … (recursive; OCR when tesseract is installed) |
| Letter | `.html` / `.eml` / `.mbox`, or a folder of them |
| PDF | `.pdf` (text via `pdftotext` when installed; otherwise filename) |

## Privacy

No telemetry. `folio serve` binds localhost only. Folio only sees files you
hand it (or drop into the reading room). See [SECURITY.md](SECURITY.md).

## Status

v0.6 · heading to **v0.57** public launch · MIT
