# ydb-installer

![YDB](web/logo-full-site-header.svg)

YDB Installer is a tool for installing YDB cluster for developers, devops, reseachers. It uses ansible collections.

## Quick start

### Prerequisites

- Go ≥ 1.20
- Node.js ≥ 18 + npm

### Build the binary

```bash
make build         # installs UI deps, builds SPA into internal/embed/dist, then go build
./ydb-installer    # serves the app on http://127.0.0.1:8080
```

Flags: `--port 8080`, `--host 127.0.0.1`, `--no-browser`.

### Develop with HMR

```bash
make dev           # runs Vite (5173) + Go (8080) in parallel; Ctrl+C stops both
```

Open http://localhost:5173 — the dev server proxies `/api/*` and `/ws/*`
to the Go backend.

### Run tests

```bash
make test          # go test (race) + vitest run
make test-go
make test-ui
```
