# folio

> Chats. Screenshots. Newsletters. Searchable. On your disk.

A single-binary local library. WhatsApp gave you a zip. Your phone gave you
10,000 screenshots. Substack gave you an inbox. folio makes them findable
without an account, a cloud, or Docker.

```
folio ingest chat  WhatsApp\ Chat.zip
folio ingest shots ~/Desktop/Screenshots
folio ingest letter weekly-dispatch.eml
folio search "boarding pass"
folio serve
```

Data lives at `~/.folio/folio.db`. Nothing leaves the machine.

## What it ingests

1. **Chat** — official WhatsApp `_chat.txt` or the zip WhatsApp emails you.
   We never touch the encrypted database.
2. **Shots** — png/jpg/webp in a folder. Tesseract OCR if installed; otherwise
   the filename is indexed so you still have a trail.
3. **Letter** — a newsletter `.html` or `.eml`. Scripts stripped.

## Privacy

No telemetry. Bind is localhost-only. Folio is the opposite of Rewind:
it only sees files you hand it.

## Status

Private v0.1. MIT.
