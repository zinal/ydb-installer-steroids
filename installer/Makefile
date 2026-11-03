.PHONY: help build dev test test-go test-ui coverage coverage-go coverage-ui clean install-ui run runtime runtime-force runtime-clean artefacts-clean

PLATFORM ?=

help:
	@echo "Targets:"
	@echo "  make install-ui    — npm install in ui/"
	@echo "  make dev           — run Vite + Go in parallel for development"
	@echo "  make runtime       — build the embedded Python+Ansible runtime archive"
	@echo "                       (reused from artefacts/<PLATFORM>/ cache when present)"
	@echo "  make runtime-force — force a fresh runtime build, ignoring the artefacts/ cache"
	@echo "  make build         — build runtime (if needed) + SPA + Go binary (./ydb-installer)"
	@echo "  make run           — run the binary (after build)"
	@echo "  make test          — run all unit tests (Go + Vitest)"
	@echo "  make test-go       — run only Go unit tests"
	@echo "  make test-ui       — run only Vitest"
	@echo "  make coverage      — run all coverage reports (Go + Vitest)"
	@echo "  make coverage-go   — Go coverage profile + per-function table + HTML"
	@echo "  make coverage-ui   — Vitest V8 coverage report"
	@echo "  make runtime-clean — remove the built runtime archive (restore placeholder)"
	@echo "  make artefacts-clean — remove the local runtime build cache (artefacts/)"
	@echo "  make clean         — remove built artifacts"

install-ui:
	cd ui && npm install

dev:
	bash scripts/dev.sh

runtime:
	PLATFORM=$(PLATFORM) bash scripts/build-runtime.sh

# FORCE_RUNTIME_BUILD=1 bypasses the artefacts/ cache and rebuilds from scratch.
runtime-force:
	PLATFORM=$(PLATFORM) FORCE_RUNTIME_BUILD=1 bash scripts/build-runtime.sh

build:
	PLATFORM=$(PLATFORM) bash scripts/build.sh

run: build
	./ydb-installer

test: test-go test-ui

test-go:
	go test ./test/unit/... ./internal/... -race

test-ui:
	cd ui && npm run test:run

coverage: coverage-go coverage-ui

coverage-go:
	go test ./test/unit/... ./internal/... -coverprofile=coverage.out -coverpkg=./internal/...
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o coverage-go.html

coverage-ui:
	cd ui && npm run test:coverage

runtime-clean:
	rm -f internal/runtime/assets/runtime-linux-amd64.tar.gz
	rm -rf .runtime-build
	printf 'placeholder runtime archive\n' > /tmp/PLACEHOLDER && \
		tar -czf internal/runtime/assets/runtime-linux-amd64.tar.gz -C /tmp PLACEHOLDER && \
		rm -f /tmp/PLACEHOLDER
	printf '{\n  "embedded": false,\n  "python": "",\n  "ansible": "",\n  "ansible_core": "",\n  "collection": "",\n  "em_collection": "",\n  "sha256": "",\n  "collections_subdir": "collections",\n  "root_subdir": "python"\n}\n' \
		> internal/runtime/assets/runtime.version

artefacts-clean:
	rm -rf artefacts

clean:
	rm -f ydb-installer
	rm -f coverage.out coverage-go.html
	rm -rf ui/coverage
	rm -rf internal/embed/dist
	mkdir -p internal/embed/dist
	touch internal/embed/dist/.gitkeep
	rm -rf .runtime-build
