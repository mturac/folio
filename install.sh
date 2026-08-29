#!/usr/bin/env bash
# Install folio into a user bin directory. Prefers a released binary when
# available; otherwise builds with Go.
set -euo pipefail

REPO="${FOLIO_REPO:-mturac/folio}"
BIN_DIR="${FOLIO_BIN_DIR:-$HOME/.local/bin}"
mkdir -p "$BIN_DIR"

have() { command -v "$1" >/dev/null 2>&1; }

install_from_go() {
  if ! have go; then
    echo "folio: need Go (https://go.dev/dl/) or a release binary." >&2
    exit 1
  fi
  echo "Installing with go install …"
  GOBIN="$BIN_DIR" go install "github.com/${REPO}@latest"
}

install_from_release() {
  local tag os arch asset url tmp
  tag="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1)"
  if [[ -z "$tag" ]]; then
    return 1
  fi
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64) arch=amd64 ;;
    aarch64|arm64) arch=arm64 ;;
    *) return 1 ;;
  esac
  case "$os" in
    linux|darwin) ;;
    *) return 1 ;;
  esac
  asset="folio_${tag#v}_${os}_${arch}.tar.gz"
  url="https://github.com/${REPO}/releases/download/${tag}/${asset}"
  tmp="$(mktemp -d)"
  if ! curl -fsSL "$url" -o "$tmp/folio.tgz"; then
    rm -rf "$tmp"
    return 1
  fi
  tar -xzf "$tmp/folio.tgz" -C "$tmp"
  install -m 755 "$tmp/folio" "$BIN_DIR/folio"
  rm -rf "$tmp"
  echo "Installed $BIN_DIR/folio ($tag)"
}

if ! install_from_release; then
  install_from_go
fi

case ":$PATH:" in
  *":$BIN_DIR:"*) ;;
  *)
    echo
    echo "Add to your PATH:"
    echo "  export PATH=\"$BIN_DIR:\$PATH\""
    ;;
esac

echo
echo "Try:"
echo "  folio"
echo "  folio init"
echo "  folio serve --open"
