# Security

Folio is a local library. Threat model and promises:

## Promises

- No telemetry, analytics, or phone-home.
- `folio serve` binds **loopback only** (`127.0.0.1` / `::1` / `localhost`).
  Non-loopback hosts are refused.
- Ingest only reads paths you pass on the CLI. It does not scan your disk
  unless you point `watch` or `ingest shots` at a directory.
- WhatsApp: only official export `_chat.txt` (or zip containing it). Folio
  never opens the encrypted WhatsApp database.
- FTS queries are sanitized against FTS5 operator injection.
- Zip members are allowlisted by chat export filename.

## Media endpoint

`GET /api/media?id=` serves a screenshot file only when:

1. the item exists and `kind=shot`, and
2. the request is on the local reading room (loopback bind).

Treat `folio serve` like opening a folder on your machine: anyone with
local access to that port can read library content.

## Reporting

Open a private GitHub security advisory on this repository, or email the
maintainer listed on the GitHub profile. Please do not file public issues
for unfixed vulnerabilities.
