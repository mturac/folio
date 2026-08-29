# Roadmap

Public launch target: **v0.57.0**.

Folio stays a single binary. Each minor version should make the product
easier to hold, not broader for its own sake.

## Version train

| Tag | Focus |
|-----|--------|
| **0.2** | Library core (ingest / search / reading room) — merged |
| **0.3** | First-run, install script, drop-to-ingest, `serve --open` — merged |
| **0.4** | Message-level chat index (long exports stay sharp) — merged |
| **0.5** | GitHub Release binaries (macOS / Linux / Windows) — merged |
| **0.6–0.15** | Format depth (Signal export, PDF text, richer EML) |
| **0.16–0.30** | Reading-room polish, TR/EN copy, keyboard flows |
| **0.31–0.45** | Watch reliability, backup/export UX, doctor depth |
| **0.46–0.56** | Release hardening, signed binaries, docs pass |
| **0.57** | Public launch tag |

Numbers move when the focus ships — empty bumps are not the point.

## Now

- [x] Clean history on main (no Cursor authorship / Co-authored-by)
- [x] Tags `v0.3.0` … `v0.6.0`
- [ ] Connect Buildkite pipeline to `mturac/folio` (see [docs/buildkite.md](docs/buildkite.md))
- [ ] First green Buildkite build on `main`
- [ ] Publish missing release archives for `v0.5.0` / `v0.6.0` via GoReleaser
- [ ] Repo visibility → **public** (owner action, when ready for OSS)

## Non-goals

- Cloud sync accounts
- Always-on screen capture
- Docker-required installs
- Reading encrypted messenger databases
