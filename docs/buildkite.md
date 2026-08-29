# Buildkite CI

Folio uses **Buildkite** for CI (same setup as miraView). GitHub Actions
is not the primary runner here.

| Step | Command |
| --- | --- |
| `test` | `go test ./...` |
| `vet` | `go vet ./...` |
| `build` | `go build` + `folio version` |

Pipeline file: [`.buildkite/pipeline.yml`](../.buildkite/pipeline.yml).

## One-time setup

### 1. Buildkite org + GitHub connection

1. Create / open a Buildkite organization.
2. Connect the GitHub App (or OAuth) so Buildkite can see `mturac/folio`.
3. **Pipelines → New pipeline** → repository `mturac/folio`.
4. Set the default branch to `main`.
5. **Steps** (YAML editor), replace with:

```yaml
steps:
  - label: ":pipeline: Upload"
    command: buildkite-agent pipeline upload .buildkite/pipeline.yml
```

### 2. Agent

Agents need Docker (pipeline steps run in `golang:1.25.0-bookworm` via the
Docker plugin).

```bash
# Example: token from Buildkite → Agents → New agent token
TOKEN=bkua_...   # do not commit; do not paste into chat logs
curl -fsSL https://raw.githubusercontent.com/buildkite/agent/main/install.sh | bash
buildkite-agent start --token "$TOKEN" --tags "queue=default,docker=true"
```

Or use Buildkite Elastic CI / hosted agents if your plan includes them.

### 3. First green build

Push to `main` or open a PR. Confirm:

1. Pipeline upload step succeeds  
2. `test` green  
3. `vet` green  
4. `build` green  

### 4. Releases

Cross-platform binaries still come from GoReleaser (`.goreleaser.yml`).
Until a Buildkite release step is wired with a GitHub token, you can publish
from a machine that has goreleaser + `GITHUB_TOKEN`:

```bash
git checkout v0.6.0   # or the tag you want
goreleaser release --clean
```

Keep agent / API tokens in Buildkite secrets or your agent host — never commit them.

## Local dry-run

```bash
make test
make vet
make build
```
