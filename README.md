# folio

> Chats. Screenshots. Newsletters. Searchable. On your disk.

A single-binary local library. WhatsApp gave you a zip. Your phone gave you
10,000 screenshots. Substack gave you an inbox. folio makes them findable
without an account, a cloud, or Docker.

```
go install github.com/mturac/folio@latest

folio ingest chat  WhatsApp\ Chat.zip
folio ingest chat  telegram-export.html
folio ingest shots ~/Desktop/Screenshots
folio ingest letter weekly-dispatch.eml
folio search "boarding pass"
folio stats
folio serve
```

Data lives at `~/.folio/folio.db`. Nothing leaves the machine.

## Install

**Requires Go 1.22+.**

```bash
go install github.com/mturac/folio@latest
```

Or from a clone:

```bash
git clone https://github.com/mturac/folio
cd folio && make install
```

Optional: install [Tesseract](https://tesseract-ocr.github.io/) for screenshot OCR.
Without it, filenames are still indexed. `eng+tur` is preferred when both
traineddata packs are present.

```bash
# macOS
brew install tesseract tesseract-lang
# Debian/Ubuntu
sudo apt install tesseract-ocr tesseract-ocr-eng tesseract-ocr-tur
```

## What it ingests

| Kind | Command | Accepts |
|------|---------|---------|
| **Chat** | `folio ingest chat <path>` | WhatsApp `_chat.txt` / zip, Telegram Desktop HTML or `DD.MM.YYYY` text |
| **Shots** | `folio ingest shots <dir>` | png/jpg/webp/… recursively; re-ingest refreshes OCR |
| **Letter** | `folio ingest letter <path>` | `.html` / `.eml` / `.mbox`, or a folder of them |

```
folio watch shots ~/Desktop/Screenshots   # re-ingest on change (poll)
folio export md > library.md
folio doctor
folio rm 42                               # or folio rm /path/to/source
folio version
```

## Reading room

`folio serve` opens a localhost-only UI: live search, kind filters, detail
sheet, and inline screenshot images. Bind refuses anything but loopback.

## Privacy

No telemetry. No accounts. Bind is localhost-only. Folio is the opposite of
Rewind: it only sees files you hand it. See [SECURITY.md](SECURITY.md).

## Status

v0.2 · MIT · built to stay small.
