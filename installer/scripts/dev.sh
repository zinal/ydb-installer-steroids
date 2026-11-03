#!/usr/bin/env bash
# Dev runner — Vite dev server + Go API server, both with live reload-ish workflow.
#
#   * Vite serves the SPA on :5173 and proxies /api and /ws to :8080.
#   * Go listens on :8080.
#
# Use Ctrl+C to stop both.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

cleanup() { kill 0; }
trap cleanup EXIT

echo "==> Starting Go backend on :8080"
go run ./cmd/installer --port 8080 &

echo "==> Starting Vite dev server on :5173"
(cd ui && npm install --silent && npm run dev) &

wait
