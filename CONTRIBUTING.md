# Contributing

Folio stays a single binary. Prefer stdlib. New dependencies need a clear
reason in the PR description.

## Dev loop

```bash
make test
make vet
go run . ingest chat testdata/chat_whatsapp.txt
go run . serve
```

## Guidelines

1. **Local-first** — no network calls in ingest/search/serve except binding
   loopback HTTP. No telemetry.
2. **Fail closed** — FTS input is sanitized; zip only accepts official chat
   names; listen refuses non-loopback hosts.
3. **Tests first for parsers** — every ingest format gets a fixture under
   `testdata/` and a table/unit test.
4. **Re-ingest is upsert** — same `source` updates body/OCR; never silently
   duplicate.
5. **Small diffs** — one concern per PR when possible.

## CI

Pull requests run `go test` / `go vet` on GitHub Actions (`.github/workflows/ci.yml`).

## PR checklist

- [ ] `make test` and `make vet` pass
- [ ] New behavior covered by a test
- [ ] README / usage updated if CLI surface changed
