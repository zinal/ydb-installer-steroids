# AGENTS.md — YDB Installer

Single source of truth for AI assistants. Read this first.

---

## 1. DEFINITION OF DONE

Every change MUST satisfy all four items before completion:

1. **i18n** — every user-visible string uses `t('key')`; keys added to BOTH `ui/src/locales/en.json` AND `ui/src/locales/ru.json`. No hardcoded UI strings.
2. **State** — UI state lives in Pinia stores (`ui/src/stores/`). Never store UI state in components.
3. **Inventory sync** — every field mapping to `inventory.yaml` MUST sync bidirectionally via `useInventorySync` (50-inventory.yaml).
4. **Tests** — new/changed logic covered per §4 coverage rules; `make test` passes.

Workflow: **read** relevant files → (optional conversation with user) → **plan** (todo list) → **implement** → **test** (`make test`) → **report** (in chat: short "how to manual verify" steps + a one-line ready-to-copy commit message) → **update this file** (with user permission).

---

## 2. BUILD & TEST (always via Makefile)

| Command | Purpose |
|---------|---------|
| `make build` | Full build (runtime + Vue + Go) |
| `make test` | ALL tests (Go + Vitest) — must pass before completion |
| `make test-go` | Go tests only (`./test/unit/... ./internal/...`) |
| `make test-ui` | Vitest only (`cd ui && npm run test:run`) |
| `make coverage` | Full coverage (Go + Vitest) — produces `coverage.out` + `coverage-go.html` + `ui/coverage/` |
| `make coverage-go` | Go coverage via `-coverpkg=./internal/...` (per-function table + HTML) |
| `make coverage-ui` | Vitest V8 coverage (`npm run test:coverage` → text + HTML + lcov) |
| `make dev` | Dev mode (Vite + Go parallel) |

---

## 3. CODING STANDARDS

### Frontend (Vue 3 + Vuetify + Pinia + vue-i18n)
- **Reuse first**: check `ui/src/components/` before creating a component.
- **i18n**: `t('key')` only; both locales updated together.
- **State**: Pinia stores only; components hold no UI state.
- **Inventory sync**: bidirectional via the two sync composables (§1.3).
- **Layout**: all screens share one centered column — `--content-max-width`/`--content-gutter` tokens + `.layout-column` in `ui/src/styles/app.css`. Never add per-screen `max-width` wrappers. Header stays full-width.
- **Step numbering**: user-visible numbers derive from `visibleFlowSteps` via `ScreenEyebrow` + `session.currentStepNumber`. Never hardcode screen IDs; numbers are sequential per flow.

### Reactive hydration rule (IMPORTANT)
Never hydrate a local ref from a Pinia store in a one-shot `onMounted`: App.vue's sync composables populate stores AFTER child `onMounted`, so one-shot reads see empty stores and forms silently keep defaults. Use a guarded `watch(..., { immediate: true })` with a single-shot flag (e.g. `let hydrated = false`). Applied in: HostsScreen, DisksScreen, TLSScreen.

### Backend (Go)
- **Handlers**: parse request → delegate to service → write JSON response.
- **Services**: stateless, testable; dependencies via interfaces (see `ansible.Runtime`).

### Comments — "why, not what"
Comments explain **why**, never **what**. Remove anything the code already says.
- **Keep**: GoDoc on exported symbols (trimmed), pitfall/gotcha warnings, non-obvious design decisions, complex business logic rationale.
- **Remove**: section banners (`// ── … ──`, `/* ── … ── */`), obvious "what" comments (`// Read the file`), getter/setter restatements, verbose GoDoc where the name is self-documenting, HTML/CSS section labels in Vue.
- **Tests**: keep GoDoc that names the scenario; remove inline step comments.

---

## 4. TESTS

**Discovery**: test files mirror source names — `ls test/unit/` and `ui/tests/` before adding; do not re-list them here. Tests that need a real inventory parse a controlled fixture in `test/testdata/`, never the live `inventory/` directory.

### Go (`test/unit/`, package `unit_test`)
- Standard `testing` only; no frameworks, assertion libs, or mock generators.
- Naming `Test<Type>_<Scenario>`; one scenario per test; table-driven for variants.
- `t.Helper()` for setup; `t.Error`/`t.Fatal` directly; YAML/JSON as raw literals.
- Exceptions: `internal/runtime/`, `internal/services/ansible/` co-locate `_test.go`.

### Vue (`ui/tests/<name>.spec.ts`)
- Vitest + `@vue/test-utils`; Vuetify + polyfills auto-configured via `setup.ts`.
- Store tests: `setActivePinia(createPinia())` in `beforeEach`; test state/getters/actions directly.
- Component tests: `mount()`; `vi.mock()` for API client, `vi.fn()` for callbacks.

### Coverage (mandatory)
Every new store action/getter, API handler, service function, and every component with conditional rendering or user interaction MUST have a test.

**Exact coverage measurement:**
- **Go**: tests live in package `unit_test` → plain `-cover` measures the test package. Use `-coverprofile=coverage.out -coverpkg=./internal/...` to instrument production code instead; `go tool cover -func` gives per-function % + total; `-html` produces a line-by-line report. Excludes `cmd/installer/` (wiring) and legacy `main.go`+`web/`.
- **UI**: Vitest needs `@vitest/coverage-v8@^2.1.3` (version-paired); `include: ['src/**/*.{ts,vue}']` excluding `main.ts`/plugins/types. Reporters `text` (per-file Stmts/Branch/Funcs/Lines + total), `html`, `lcov`. Thresholds set at the achieved level in `vitest.config.ts` to block regressions; run `make coverage` for both reports.

---

## 5. PROJECT MAP

**Entry point**: `cmd/installer/main.go` (flags, `setupRuntime()`, `OperationStates`, router, inventory path discovery, startup banner). `internal/version/version.go` is the single version source; `cmd/installer/version.go` aliases it, and the session API exposes it to the UI.
SPA is embedded via `internal/embed/embed.go` (`embed.FS` + `WriteAnsibleCfg()` fallback).

### Backend — API (`internal/api/`)
| File | Endpoints | Purpose |
|------|-----------|---------|
| `router.go` | `GET /api/healthz` | Route registration, `Options` struct, SPA fallback, flow-aware wiring |
| `middleware.go` | — | CORS, request logging; `Flush()` on `statusRecorder` required for SSE streaming |
| `handlers/session.go` | `GET/PUT /api/session` | Flow, steps, language, `ui_state` (navigation history) |
| `handlers/inventory.go` | `GET/PUT /api/inventory`, `POST /api/inventory/validate`, `GET /api/system/configs` | 50-inventory.yaml; PUT re-syncs session DynamicNodes |
| `handlers/inventory_ws.go` | `WS /ws/inventory` | Inventory change pub/sub |
| `handlers/hosts.go` | `POST /api/hosts/check` | SSH + Python + CPU + ports + NTP |
| `handlers/disks.go` | `POST /api/disks/check`, `POST /api/disks/validate` | lsblk discovery; manual path check |
| `handlers/deps.go` | `GET /api/system/deps`, `POST /api/system/deps/install` | Bundled flag; system mode probes python, ansible, and `ydb_platform.ydb` |
| `handlers/runtime.go` | `GET /api/system/runtime` | Embedded runtime info |
| `handlers/connectivity.go` | `GET /api/system/connectivity` | Internet-access probe (GitHub API, short timeout); `ConnectivityAPIURL` var for test injection |
| `handlers/versions.go` | `GET /api/versions`, `POST /api/versions/validate-file` | GitHub releases; local file validation; `VersionsAPIURL` var for test injection |
| `handlers/tls.go` | `POST /api/tls/generate`, `/api/tls/validate`, `/api/tls/exists` | Self-signed generation; validation |
| `handlers/ws.go` | `WS /ws/logs` | Flow-aware log WebSocket; SSE fallback when no Upgrade header |
| `handlers/install.go` | `GET /api/system/deps/instruction/:os`, `GET /api/install/ydb/config-status`, `POST /api/install/ydb/generate-config`, `POST /api/install/ydb/start`, `POST /api/install/ydb/create-database`, `POST /api/install/delete-ydb/start`, `POST /api/install/cancel`, `GET /api/install/status`, `GET /api/install/logs` | Config generation + YDB install/delete; flow-aware status/cancel/logs; `ConfigStatus` dry-run compare (`missing\|matches\|outdated`) via `Generator.Render` + session DC enrichment |

### Backend — Runtime (`internal/runtime/`)
| File | Purpose |
|------|---------|
| `manager.go` | Resolve/extract/verify; version-stamped cache; atomic rename; sha256; `YDB_INSTALLER_RUNTIME_DIR` override |
| `paths.go` | `Paths` + `ExtraEnv()` (PATH, PYTHONHOME, ANSIBLE_COLLECTIONS_PATH); satisfies `ansible.Runtime` |
| `extract.go` | tar.gz extraction: traversal-neutralization, symlink containment, exec-bit, size cap |
| `embed.go` | `//go:embed` archive; `ReadMetadata()`, `VerifyArchiveSHA()` |
| `assets/` | `runtime-linux-amd64.tar.gz` + `runtime.version` (placeholder or real) |

### Backend — Banner (`internal/banner/`)
| File | Purpose |
|------|---------|
| `banner.go` | Boxed ANSI startup banner on stdout: URL (bold green) + working-directory notice (bright yellow); `NO_COLOR`/TTY-aware; `Render`/`RenderPlain` pure functions |

### Backend — Services (`internal/services/`)
| File | Purpose |
|------|---------|
| `checker/deps.go` | Probe python/ansible/`ydb_platform.ydb`; system mode: 3 deps |
| `checker/instruction.go` | OS install instructions; git URL for `ydb_platform.ydb` |
| `config/watcher.go` | Discover 50-inventory.yaml + config.yaml at startup |
| `config/config_generator.go` | Generate config.yaml (DC-aware, block-4-2); `Render()` = in-memory dry-run; `Backup()` |
| `config/inventory.go` / `inventory_writer.go` | Parse / write 50-inventory.yaml (`WriteWithPatch()` preserves comments; mutex-serialized) |
| `config/inventory_patcher.go` | Surgical YAML Node updates; preserves comments/unknown vars; `nil` deletes key |
| `config/inventory_watcher.go` | Watch 50-inventory.yaml for external changes |
| `config/yaml_helpers.go` | Shared YAML node helpers |
| `session/store.go` | In-memory session; `UIState` navigation history |
| `lang/detect.go` | System language auto-detect |
| `hosts/checker.go` | SSH + Python + CPU + ports + NTP chain; one SSH session per host |
| `disks/checker.go` | Disk discovery (lsblk via SSH); `ValidateDisks()` |
| `sshclient/sshclient.go` | Shared SSH args; `UserKnownHostsFile=/dev/null`; ControlMaster |
| `tls/checker.go` / `tls/generator.go` | Cert validation / self-signed generation (host-conflict block; `Force` = clean-slate: wipe `TLS/CA`, fresh CA) |
| `ansible/runner.go` | Run ansible-playbook; `Runtime` interface; bundled via `<python> <ansible-playbook>`; typed-nil guard |
| `logs/broadcaster.go` / `logs/ring.go` | Pub/sub fan-out + file persistence / 10000-cap ring buffer |
| `progress/parser.go` | Derive intra-playbook progress from Ansible `TASK [` lines (`started/(started+9)` curve); fed by `InstallState.WriteLog`, blended in `recalcProgress` |

### Backend — Models (`internal/models/`)
`session.go` (Session, Flow, Step, UIState) · `install.go` (InstallState, OperationStates ×2 + `ForFlow()`) · `inventory.go` (InventoryData, TopologyConfig, DatabaseConfig, DynamicNode) · `hosts.go` · `disks.go`

### Frontend — Entry
`main.ts` (bootstrap + plugins) · `App.vue` (layout shell; wires `useInventorySync`, `useSessionSync`, `usePersistedState`, `useRuntimeStore.load()`)

### Frontend — Screens (`ui/src/views/`)
| ID | Screen | File | Note |
|----|--------|------|------|
| 1 | Introduction | `IntroScreen.vue` | Mode select, deps check, connectivity card (status + Offline mode checkbox + help tooltip); delete-ydb needs inventory+config; a single visible card fills its grid row |
| 2 | Topology | `TopologyScreen.vue` | Cards + diagrams; per-topology state preservation |
| 3 | Hosts & SSH | `HostsScreen.vue` | DC host tables, SSH, file import, NTP banner |
| 4 | Disks | `DisksScreen.vue` | Auto/manual mode, validation, size warnings; language-aware system-requirements docs link in header |
| 5 | Distribution | `DistributionScreen.vue` | Official/local source, GitHub versions; offline mode auto-switches to local + disables the official tab |
| 6 | Configuration | `YdbConfigScreen.vue` | Domain, password, cores, booleans, CoresInfoCard |
| 7 | Databases | `DatabasesScreen.vue` | Variants single / multiple |
| 8 | TLS | `TLSScreen.vue` | Generate/existing |
| 9 | Installation | `InstallationScreen.vue` | Config status chip (matches/outdated/missing); `InstallProgressPanel` |
| 10 | Result | `ResultScreen.vue` | `ResultSummary`; YDB UI link |
| 30/31 | Delete YDB | `DeleteYdbConfirmScreen` / `DeleteYdbProgressScreen` | DELETE confirmation; recovery-only progress |
| * | Placeholder | `PlaceholderScreen.vue` | Stub |

Router: `ui/src/router/index.ts` — `REAL_VIEWS` mapping + placeholder bypass.

### Frontend — Stores (`ui/src/stores/`)
`session.ts` (flow, steps, statuses, `beforeNext` gate, `currentStepNumber`) · `topology.ts` · `hosts.ts` (SSH + DC hosts) · `disks.ts` · `distribution.ts` · `ydbConfig.ts` · `databases.ts` (variants, duplicate detection) · `tls.ts` · `runtime.ts` (`bundleActive`) · `connectivity.ts` (status/override/`offlineMode`/`effectiveOnline`; probe via `GET /api/system/connectivity`) · `ui.ts` (console collapse toggle, persisted)

### Frontend — Composables (`ui/src/composables/`)
`useInventorySync.ts` (two-way 50 sync: debounced PUT + WS reload; dirty-flag trailing flush; app-level singleton — App.vue only) · `useSessionSync.ts` (UI→backend navigation history) · `usePersistedState.ts` (sessionStorage ephemerals) · `useInstall.ts` (install lifecycle; `configStatus` ref; always calls API) · `useDeps.ts` · `useCoresInfo.ts` · `useHostChecks.ts` · `useDiskChecks.ts` · `useHostFileImport.ts`

### Frontend — Components (`ui/src/components/`)
- `layout/`: `AppHeader` (logo, shared installer version badge, language switch), `AppSidebar` (steps + status icons), `AppFooter` (Back/Next; awaits `beforeNext`)
- `common/`: `AppVersion` (session-backed installer version + localized `BETA` badge), `ScreenEyebrow` (step pill, danger variant), `InstallProgressPanel` (screens 9/31; collapsible console, persisted), `ResultSummary` (screen 10), `DepsPanel` (bundled banner + CLI hint), `ConsoleOutput`, `TaskList`, `ProgressBar`, `CoresInfoCard`, `HelpTooltip`, `RestartButton`, `ConfirmDialog`
- `forms/SshSettings` · `hosts/DCHostInput` · `disks/DiskCard` + `DiskTable` · `topology/TopologyCard` + `TopologyDiagram` · `checks/CheckResultCard` + `CheckResultsList`

### Frontend — Config / API / Types / Locales
`config/flows.ts` (single source of truth for step IDs) · `config/operations.ts` (flow → endpoint/screens/i18n) · `config/topologies.ts` · `config/helpLinks.ts` (doc URLs; `systemRequirements` stores a locale-less path, `systemRequirementsUrl(locale)` builds the `/ru|en/` URL) · `api/client.ts` (typed HTTP/WS + `ConfigMatchStatus`) · `types/*.ts` · `plugins/i18n.ts` + `plugins/vuetify.ts` · `locales/{en,ru}.json` · `utils/sshArgs.ts` · `styles/app.css` (design tokens + `.layout-column`)

### Build & Scripts
`Makefile` · `scripts/build.sh` (runtime + Vue + Go) · `scripts/build-runtime.sh` · `scripts/runtime.lock` (pinned versions + `ydb_platform.ydb` git URL/ref; local fallback path) · `scripts/dev.sh`

---

## 6. ARCHITECTURE

```
Browser (Vue SPA) ←→ Go HTTP (embed.FS)
     ↓                    ↓
  Pinia stores      Handlers → OperationStates → Services → FS
                          ↓
                 RuntimeManager (Python+Ansible) → ansible-playbook
                 InstallState ×2: Install / DeleteYdb
```
Embedded runtime bundles `ydb_platform.ydb`. `Paths.ExtraEnv()` sets `ANSIBLE_COLLECTIONS_PATH` so the collection resolves without host Ansible config.

### Flows
Defined in `ui/src/config/flows.ts` (source of truth): `ydb` = 1,2,3,4,5,6,7,8,9,10 · `delete-ydb` = 1,30,31.

### Inventory sync (bidirectional)
```
50-inventory.yaml:  store change → useInventorySync (500ms debounce; dirty-flag + trailing flush) → PUT /api/inventory
                      → InventoryWriter.WriteWithPatch() (preserves comments)
                    external edit → InventoryWatcher → WS /ws/inventory → GET + apply to stores
```

### State persistence (survives refresh)
| Level | Storage | Content | Mechanism |
|-------|---------|---------|-----------|
| Domain data | inventory.yaml (50) | hosts, SSH, disks, distribution, ydb_config, DBs, TLS path, topology | `useInventorySync` |
| Navigation | Backend session | currentStep, visited, errored, configGenerated | `useSessionSync` (debounced PUT) |
| Ephemerals | sessionStorage | disks entry mode, TLS mode/days, check results, per-variant DB saves, connectivity `override` (offline mode), console collapse toggle | `usePersistedState` (schema-versioned) |

Load order: router `beforeEach` (URL → currentStep/visited) → `usePersistedState` rehydrate → `session.load()` MERGES backend `ui_state` (adopts backend currentStep only if still default '1') → sync subscriptions start.

### Operations (each flow owns one InstallState)
| Flow | State | Start endpoint | Playbooks |
|------|-------|----------------|-----------|
| ydb | `Install` | `POST /api/install/ydb/start` | `ydb_platform.ydb.initial_setup` → `install_dynamic` for extra databases |
| delete-ydb | `DeleteYdb` | `POST /api/install/delete-ydb/start` | `ydb_platform.ydb.format_cluster` |

Status/cancel/logs (`GET /api/install/status`, `POST /api/install/cancel`, `GET /api/install/logs`, `WS /ws/logs`) are flow-aware via `OperationStates.ForFlow(session.Flow)`. `ConfigStatus` dry-run: `Generator.Render` + session DC enrichment vs on-disk config.yaml → `missing|matches|outdated`. Create-database: position-based dynnode letters (a,b,c…).

### Embedded runtime
`build-runtime.sh`: python-build-standalone + pip ansible + `ydb_platform.ydb` from git `main` → tar.gz. The collection uses git-first-then-local-fallback (`COLLECTION_GIT_URL` → `COLLECTION_LOCAL_PATH`); Ansible installed once (cache-miss block only). The platform-keyed `artefacts/` cache does not track the floating git ref, so use `make runtime-force` (or clear the matching cache) to pick up upstream changes. Run: `Manager.Resolve()` → cache hit or sha256-verify + atomic extract → `Paths.ExtraEnv()`. Fallback: `--use-system-deps` or placeholder → system `ansible-playbook`; deps endpoint then probes 3 system deps.

### GOTCHAS (hard-won rules — do not regress)
- **Hydration race**: see §3; never one-shot `onMounted` reads of async stores.
- **Flow isolation**: separate `InstallState` per flow; always look up via `ForFlow()`; never share state across flows.
- **Delete gates**: delete-ydb selectable only with inventory+config found.
- **Multiple-DB availability**: the multiple-DB variant is always available and works with both bundled and system Ansible; never gate it behind `--use-system-deps`. The writer emits `ydb_databases` for the multiple variant.
- **Typed-nil guard**: `NewRunnerWithRuntime` must normalize typed-nil runtime (see runner_test.go).
- **Config status**: `fetchConfigStatus` always calls the API — no `configGenerated` short-circuit.
- **SSE Flusher middleware**: `statusRecorder` must implement `Flush()` (delegate to `http.Flusher` on the underlying writer) — without it SSE fallback (`/ws/logs` without Upgrade header) returns 500. Fixed in `middleware.go`.
- **parseJumpHost Jinja2**: `parseJumpHost()` regex `\S+` before `@` only captures the closing `}}` of `{{ ansible_user }}` (space-separated from variable name) → username ends up empty. Fixed in `ui/src/utils/sshArgs.ts` with explicit template pattern. The test was silently skipped (lived in `src/utils/__tests__/`, not `tests/`).
- **TLS `Force` = clean-slate**: `tls.Generate` with `Force` wipes the whole `TLS/CA` and always issues a fresh CA (never reuses); orphaned per-host dirs are gone. Non-force still reuses CA and blocks on host conflicts.
- **mirror-3-dc disk minimum**: `minDisksToSelect()` in `ui/src/config/topologies.ts` requires 3 disks below 9 hosts and 1 disk at 9+ hosts (both auto and manual entry); every other topology needs 1. `buildMirror3DCFailDomains` keeps the 9-vdisk/3-fail-domain invariant and, for N≥9, pins every host to the FIRST selected disk (`/dev/disk/by-partlabel/ydb_disk_1`). Keep the threshold (9) in sync on both sides.
- **Console collapse is shared UI state**: the console expand/collapse toggle lives in the `ui` store (`consoleExpanded`, persisted in sessionStorage) and is shared by `InstallProgressPanel` and the success console cards — toggling on one install screen affects all others.
- **Inventory sync never drops writes**: `useInventorySync` uses a dirty-flag + trailing-flush scheduler — a store mutation during an in-flight PUT re-marks dirty and triggers a second flush, so the newest state always lands. Watchers must subscribe with `{ flush: 'sync' }`: Pinia's default `'pre'` defers callbacks past the reload's `applyToStores`, so the `applying` guard would be released too early and reload-induced mutations would echo back as writes.
- **Sync composables are app-level singletons**: `useInventorySync` must be called ONLY in `App.vue`. A screen-level call creates a second instance with its own watchers, debounce timer, `isSyncing` flag and WebSocket → double PUTs, concurrent writes, and unmount cancels pending writes when navigating away.
- **updateDCHost allows colliding edits**: editing a host to a value already present in the same DC lands in the store; duplicates are surfaced by the `duplicates` getter + UI warning, never silently dropped.
- **Inventory writes are mutex-serialized**: `WriteWithPatch` holds a per-writer `sync.Mutex` so concurrent PUTs cannot interleave a read-modify-write cycle.

---

## 7. QUICK REFERENCE

**Add a screen**: create `ui/src/views/<Name>Screen.vue` → add to `REAL_VIEWS` in `router/index.ts` → store if needed → i18n keys in BOTH locales → update §5 Screens here → `make test`.

**Add an API endpoint**: handler in `internal/api/handlers/` → register in `router.go` → service if needed → update §5 API table → `make test`.

**Add a long-running operation**: playbook list in `install.go` → new `InstallState` + `ForFlow()` case → start handler via `runPlaybookSequence` (`playbookStep.ExtraVars` for extra vars) → route in `router.go` → descriptor in `ui/src/config/operations.ts` → `useInstall.startXxx()` → consume `InstallProgressPanel` → `make test`.

**Rebuild runtime**: edit `scripts/runtime.lock` → `make runtime && make build` → verify `./ydb-installer --extract-runtime-only` → `make test`.
