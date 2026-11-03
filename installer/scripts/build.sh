#!/usr/bin/env bash
# Build the YDB Installer single binary.
#
#   0. Build the embedded Python+Ansible runtime archive (unless skipped).
#   1. Install UI dependencies.
#   2. Build the SPA into internal/embed/dist (configured in ui/vite.config.ts).
#   3. Build the Go binary.
#   4. Restore the placeholder runtime assets (so the large ~100MB archive
#      never lingers in the working tree / git after the build).
#
# Set SKIP_RUNTIME_BUILD=1 to skip the (slow, network-dependent) runtime build
# and embed whatever runtime archive is already present (e.g. the placeholder).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# restore_placeholder resets the committed tiny placeholder runtime archive and
# metadata (embedded:false) to their exact committed bytes. Called after
# `go build` so the multi-megabyte real runtime — which `make runtime` wrote
# into internal/runtime/assets/ — does not linger in the working tree and risk
# an accidental commit.
#
# We restore from git (not by regenerating the tar.gz) so the placeholder is
# byte-identical to the committed version (regenerating would differ due to
# gzip/tar timestamps and show up as a spurious `git diff`).
restore_placeholder() {
  local assets="internal/runtime/assets"
  echo "==> Restoring placeholder runtime assets"
  # Prefer restoring the exact committed bytes from git.
  if git rev-parse --git-dir >/dev/null 2>&1; then
    git checkout HEAD -- "$assets/runtime-linux-amd64.tar.gz" "$assets/runtime.version" 2>/dev/null || true
  fi
  # Fallback (no git / files not tracked yet): regenerate a minimal placeholder.
  if [[ ! -f "$assets/runtime-linux-amd64.tar.gz" ]] || \
     [[ $(wc -c < "$assets/runtime-linux-amd64.tar.gz") -gt 1000000 ]]; then
    rm -f "$assets/runtime-linux-amd64.tar.gz"
    local tmpdir; tmpdir="$(mktemp -d)"
    printf 'placeholder runtime archive\n' > "$tmpdir/PLACEHOLDER"
    tar -czf "$assets/runtime-linux-amd64.tar.gz" -C "$tmpdir" PLACEHOLDER
    rm -rf "$tmpdir"
    cat > "$assets/runtime.version" <<'JSON'
{
  "embedded": false,
  "python": "",
  "ansible": "",
  "ansible_core": "",
  "collection": "",
  "em_collection": "",
  "sha256": "",
  "collections_subdir": "collections",
  "root_subdir": "python"
}
JSON
  fi
}

if [[ "${SKIP_RUNTIME_BUILD:-0}" == "1" ]]; then
  echo "==> Skipping runtime build (SKIP_RUNTIME_BUILD=1)"
  RUNTIME_BUILT=0
else
  echo "==> Building embedded runtime"
  bash scripts/build-runtime.sh
  RUNTIME_BUILT=1
fi

echo "==> Installing UI dependencies"
(cd ui && npm install)

echo "==> Building UI"
(cd ui && npm run build)

echo "==> Building Go binary"
if [[ "${PLATFORM:-}" == "aarch64-apple-darwin" ]]; then
  GOOS=darwin GOARCH=arm64 go build -o ydb-installer ./cmd/installer
else
  go build -o ydb-installer ./cmd/installer
fi

# The real runtime archive was embedded into the binary above. Restore the
# committed placeholder so the large archive does not linger in the tree.
if [[ "${RUNTIME_BUILT:-0}" == "1" ]]; then
  restore_placeholder
fi

echo "==> Done. Run ./ydb-installer"
