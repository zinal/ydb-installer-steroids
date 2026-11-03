#!/usr/bin/env bash
# build-runtime.sh — build the embedded, self-contained Ansible control-node
# runtime (relocatable CPython + Ansible + the ydb_platform collections) and pack
# it into the //go:embed asset consumed by internal/runtime.
#
# Output:
#   internal/runtime/assets/runtime-linux-amd64.tar.gz   (the runtime archive)
#   internal/runtime/assets/runtime.version              (JSON metadata)
#
# The archive layout (extracted root):
#   python/        — relocatable CPython with Ansible installed (bin/, lib/, ...)
#   collections/   — ansible_collections/ydb_platform/{ydb,ydb_em}/...
#
# Two collections are bundled:
#   ydb_platform.ydb     — YDB cluster install/delete (screens 9, 31)
#   ydb_platform.ydb_em  — YDB Enterprise Manager install (screen 24)
#
# The Python+Ansible tree is cached across invocations as long as the pinned
# versions (and PLATFORM) in runtime.lock do not change. Both collections are
# always reinstalled fresh.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# Preserve any PLATFORM / COLLECTION_LOCAL_PATH values passed by the caller
# before sourcing the lock file (which would otherwise overwrite them with
# defaults).
_PLATFORM_OVERRIDE="${PLATFORM:-}"
_COLLECTION_OVERRIDE="${COLLECTION_LOCAL_PATH:-}"

# shellcheck source=/dev/null
source scripts/runtime.lock

if [[ -n "$_PLATFORM_OVERRIDE" ]]; then
  PLATFORM="$_PLATFORM_OVERRIDE"
fi
unset _PLATFORM_OVERRIDE

if [[ -n "$_COLLECTION_OVERRIDE" ]]; then
  COLLECTION_LOCAL_PATH="$_COLLECTION_OVERRIDE"
fi
unset _COLLECTION_OVERRIDE

ASSETS_DIR="internal/runtime/assets"
ARCHIVE="$ASSETS_DIR/runtime-linux-amd64.tar.gz"
VERSION_FILE="$ASSETS_DIR/runtime.version"
WORK="$ROOT/.runtime-build"

mkdir -p "$ASSETS_DIR" "$WORK"

# Local, platform-keyed cache: a full runtime build (CPython + Ansible +
# both collections downloaded from git) takes minutes and needs network
# access. Once built for a given PLATFORM, the resulting archive is copied
# here; subsequent builds for the same PLATFORM reuse it verbatim instead of
# rebuilding, unless FORCE_RUNTIME_BUILD=1 is set or the cache is missing.
#
# This is a manual cache: it does NOT track the floating git refs
# (COLLECTION_GIT_REF/EM_COLLECTION_GIT_REF default to main/main), so
# `rm -rf artefacts/` (or FORCE_RUNTIME_BUILD=1) is required to pick up
# upstream collection changes without bumping runtime.lock.
ARTEFACTS_DIR="$ROOT/artefacts/$PLATFORM"
ARTEFACT_ARCHIVE="$ARTEFACTS_DIR/runtime-linux-amd64.tar.gz"
ARTEFACT_VERSION="$ARTEFACTS_DIR/runtime.version"

if [[ "${FORCE_RUNTIME_BUILD:-0}" != "1" ]] && \
   [[ -f "$ARTEFACT_ARCHIVE" ]] && [[ -f "$ARTEFACT_VERSION" ]]; then
  echo "==> Reusing cached runtime artefacts: $ARTEFACTS_DIR"
  cp "$ARTEFACT_ARCHIVE" "$ARCHIVE"
  cp "$ARTEFACT_VERSION" "$VERSION_FILE"
  echo "==> Done (from cache)."
  echo "    archive : $ARCHIVE ($(du -h "$ARCHIVE" | awk '{print $1}'))"
  echo "    (set FORCE_RUNTIME_BUILD=1 or remove $ARTEFACTS_DIR to rebuild)"
  exit 0
fi

PBS_BASE="https://github.com/indygreg/python-build-standalone/releases/download/${PBS_RELEASE}"
PBS_FILE="cpython-${CPYTHON_VERSION}+${PBS_RELEASE}-${PLATFORM}-install_only.tar.gz"
PBS_URL="${PBS_BASE}/${PBS_FILE}"

PYROOT="$WORK/python"
COLLECTIONS="$WORK/collections"

echo "==> Cleaning previous build tree"
# Always wipe collections (reinstalled every time); Python+Ansible tree is cached.
rm -rf "$COLLECTIONS"
mkdir -p "$WORK" "$COLLECTIONS"

PY="$PYROOT/bin/python3"

# Cache key covers everything that determines the Python+Ansible tree identity.
# If any of these change, the tree is rebuilt from scratch.
CACHE_KEY="${PLATFORM}-${CPYTHON_VERSION}-${PBS_RELEASE}-${ANSIBLE_VERSION}-${ANSIBLE_CORE_VERSION}"
CACHE_KEY_FILE="$PYROOT/.cache-key"

if [[ -f "$CACHE_KEY_FILE" ]] && [[ "$(cat "$CACHE_KEY_FILE")" == "$CACHE_KEY" ]]; then
  echo "==> Using cached Python+Ansible tree (key: $CACHE_KEY)"
else
  echo "==> Cache miss — rebuilding Python+Ansible tree"
  rm -rf "$PYROOT"

  if [[ -f "$WORK/$PBS_FILE" ]]; then
    echo "==> Using cached python-build-standalone: $PBS_FILE"
  else
    echo "==> Downloading python-build-standalone: $PBS_FILE"
    curl -fL --retry 3 -o "$WORK/$PBS_FILE" "$PBS_URL"
  fi

  echo "==> Extracting CPython"
  tar -xzf "$WORK/$PBS_FILE" -C "$WORK"
  # python-build-standalone extracts into a top-level "python/" directory.

  if [[ ! -x "$PY" ]]; then
    echo "ERROR: expected interpreter at $PY not found" >&2
    exit 1
  fi

  # echo "==> Upgrading pip"
  # "$PY" -m pip install --upgrade pip

  echo "==> Installing Ansible into the runtime"
  "$PY" -m pip install \
    "ansible==${ANSIBLE_VERSION}" \
    "ansible-core==${ANSIBLE_CORE_VERSION}"

  # Persist the cache key so subsequent runs can reuse this tree.
  echo "$CACHE_KEY" > "$CACHE_KEY_FILE"
fi

if [[ ! -x "$PY" ]]; then
  echo "ERROR: expected interpreter at $PY not found" >&2
  exit 1
fi

# Extra Python packages the bundled collections need on the control node
# (e.g. ydb_platform.ydb_em requires pyyaml). Installed into the same
# interpreter that runs ansible-playbook.
if [[ -n "${EXTRA_PYTHON_PACKAGES:-}" ]]; then
  echo "==> Installing extra Python packages: ${EXTRA_PYTHON_PACKAGES}"
  # shellcheck disable=SC2086 — intentional word splitting into separate args.
  "$PY" -m pip install ${EXTRA_PYTHON_PACKAGES}
fi

GALAXY="$PYROOT/bin/ansible-galaxy"

# ── ydb_platform.ydb (YDB cluster install/delete) ────────────────────────────
#
# Sourcing strategy (mirrors ydb_em below): prefer the pinned git URL so builds
# are reproducible; fall back to a local checkout when the URL is unset or the
# clone fails (the repository may be private with no credentials configured).
echo "==> Installing ydb_platform.ydb collection"
YDB_DEST="$COLLECTIONS/ansible_collections/ydb_platform/ydb"
YDB_SOURCE=""

export ANSIBLE_COLLECTIONS_PATH="$COLLECTIONS"

if [[ -n "${COLLECTION_GIT_URL:-}" ]]; then
  YDB_REQ_FILE="$WORK/requirements-ydb.yaml"
  cat > "$YDB_REQ_FILE" <<EOF
roles: []
collections:
  - name: ${COLLECTION_GIT_URL}
    type: git
    version: ${COLLECTION_GIT_REF:-main}
EOF
  if "$PY" "$GALAXY" collection install -r "$YDB_REQ_FILE" -p "$COLLECTIONS" --force; then
    YDB_SOURCE="git:${COLLECTION_GIT_URL}@${COLLECTION_GIT_REF:-main}"
  else
    echo "    git install failed — falling back to local checkout"
  fi
fi

# Fall back to (or start from) the local checkout when git did not deliver it.
if [[ ! -d "$YDB_DEST" ]] && [[ -n "${COLLECTION_LOCAL_PATH:-}" ]]; then
  YDB_LOCAL="$COLLECTION_LOCAL_PATH"
  [[ "$YDB_LOCAL" != /* ]] && YDB_LOCAL="$ROOT/$YDB_LOCAL"

  if [[ -f "$YDB_LOCAL/galaxy.yml" ]]; then
    echo "    installing from local path: $YDB_LOCAL"
    # Build a tarball first so build_ignore in galaxy.yml is honoured and the
    # installed tree matches what a git/Galaxy install would produce.
    YDB_BUILD_DIR="$WORK/ydb-build"
    rm -rf "$YDB_BUILD_DIR"
    mkdir -p "$YDB_BUILD_DIR"
    "$PY" "$GALAXY" collection build "$YDB_LOCAL" --output-path "$YDB_BUILD_DIR" --force
    YDB_TARBALL="$(find "$YDB_BUILD_DIR" -maxdepth 1 -name '*.tar.gz' -print -quit)"
    if [[ -z "$YDB_TARBALL" ]]; then
      echo "ERROR: failed to build the ydb collection from $YDB_LOCAL" >&2
      exit 1
    fi
    "$PY" "$GALAXY" collection install "$YDB_TARBALL" -p "$COLLECTIONS" --force
    YDB_SOURCE="path:$YDB_LOCAL"
  else
    echo "ERROR: COLLECTION_LOCAL_PATH=$YDB_LOCAL does not contain galaxy.yml" >&2
  fi
fi

unset ANSIBLE_COLLECTIONS_PATH

# The ydb collection is REQUIRED: without it the screen-9/31 playbooks cannot
# resolve, so fail loudly rather than shipping a silently broken runtime.
if [[ ! -d "$YDB_DEST" ]]; then
  echo "ERROR: ydb_platform.ydb collection was not installed into $COLLECTIONS" >&2
  echo "       Set COLLECTION_GIT_URL (with credentials) or COLLECTION_LOCAL_PATH" >&2
  echo "       in scripts/runtime.lock to a local checkout of the collection." >&2
  exit 1
fi
echo "    ydb source: ${YDB_SOURCE:-unknown}"

# ── ydb_platform.ydb_em (YDB Enterprise Manager) ────────────────────────────
#
# Required by the EM install flow: screen 24 runs
# ydb_platform.ydb_em.{initial_setup,install_agent,register_agents}.
#
# Sourcing strategy (see runtime.lock): prefer the pinned git URL so builds are
# reproducible; fall back to a local checkout when the URL is unset or the clone
# fails (the repository may be private with no credentials configured).
echo "==> Installing ydb_platform.ydb_em collection"
EM_DEST="$COLLECTIONS/ansible_collections/ydb_platform/ydb_em"
EM_SOURCE=""

export ANSIBLE_COLLECTIONS_PATH="$COLLECTIONS"

if [[ -n "${EM_COLLECTION_GIT_URL:-}" ]]; then
  EM_REQ_FILE="$WORK/requirements-em.yaml"
  cat > "$EM_REQ_FILE" <<EOF
roles: []
collections:
  - name: ${EM_COLLECTION_GIT_URL}
    type: git
    version: ${EM_COLLECTION_GIT_REF:-main}
EOF
  if "$PY" "$GALAXY" collection install -r "$EM_REQ_FILE" -p "$COLLECTIONS" --force; then
    EM_SOURCE="git:${EM_COLLECTION_GIT_URL}@${EM_COLLECTION_GIT_REF:-main}"
  else
    echo "    git install failed — falling back to the local checkout"
  fi
fi

# Fall back to (or start from) the local checkout when git did not deliver it.
if [[ ! -d "$EM_DEST" ]] && [[ -n "${EM_COLLECTION_PATH:-}" ]]; then
  # Resolve relative paths against the repo root, not the caller's CWD.
  EM_LOCAL="$EM_COLLECTION_PATH"
  [[ "$EM_LOCAL" != /* ]] && EM_LOCAL="$ROOT/$EM_LOCAL"

  if [[ -f "$EM_LOCAL/galaxy.yml" ]]; then
    echo "    installing from local path: $EM_LOCAL"
    # Build a tarball first so build_ignore in galaxy.yml is honoured and the
    # installed tree matches what a git/Galaxy install would produce.
    EM_BUILD_DIR="$WORK/em-build"
    rm -rf "$EM_BUILD_DIR"
    mkdir -p "$EM_BUILD_DIR"
    "$PY" "$GALAXY" collection build "$EM_LOCAL" --output-path "$EM_BUILD_DIR" --force
    EM_TARBALL="$(find "$EM_BUILD_DIR" -maxdepth 1 -name '*.tar.gz' -print -quit)"
    if [[ -z "$EM_TARBALL" ]]; then
      echo "ERROR: failed to build the ydb_em collection from $EM_LOCAL" >&2
      exit 1
    fi
    "$PY" "$GALAXY" collection install "$EM_TARBALL" -p "$COLLECTIONS" --force
    EM_SOURCE="path:$EM_LOCAL"
  else
    echo "ERROR: EM_COLLECTION_PATH=$EM_LOCAL does not contain galaxy.yml" >&2
  fi
fi

unset ANSIBLE_COLLECTIONS_PATH

# The EM collection is REQUIRED: without it the screen-24 playbooks cannot
# resolve, so fail loudly rather than shipping a silently broken runtime.
if [[ ! -d "$EM_DEST" ]]; then
  echo "ERROR: ydb_platform.ydb_em collection was not installed into $COLLECTIONS" >&2
  echo "       Set EM_COLLECTION_GIT_URL (with credentials) or EM_COLLECTION_PATH" >&2
  echo "       in scripts/runtime.lock to a local checkout of the collection." >&2
  exit 1
fi

# Verify the playbooks the installer actually invokes are present.
for pb in initial_setup install_agent register_agents; do
  if [[ ! -f "$EM_DEST/playbooks/$pb.yaml" ]] && [[ ! -f "$EM_DEST/playbooks/$pb.yml" ]]; then
    echo "ERROR: ydb_em collection is missing playbooks/$pb.yaml" >&2
    echo "       The EM install flow (screen 24) requires it." >&2
    exit 1
  fi
done
echo "    ydb_em source: ${EM_SOURCE:-unknown}"

echo "==> Pruning to reduce size"
# NOTE: only delete artifacts that are provably unused by Ansible at runtime.
# Do NOT blanket-remove directories named "test"/"tests" — Ansible ships a
# legitimate ansible/plugins/test/ package (AnsibleJinja2Test); deleting it
# breaks ansible.cli imports with ModuleNotFoundError.
find "$PYROOT" -type d -name "__pycache__" -prune -exec rm -rf {} + 2>/dev/null || true
find "$PYROOT" -type f \( -name "*.pyc" -o -name "*.pyo" \) -delete 2>/dev/null || true
rm -rf "$PYROOT/lib"/python*/idlelib 2>/dev/null || true
rm -rf "$PYROOT/lib"/python*/tkinter 2>/dev/null || true
rm -rf "$PYROOT/lib"/python*/ensurepip 2>/dev/null || true
rm -rf "$PYROOT/share/doc" 2>/dev/null || true
# Remove per-package test directories ONLY under site-packages dist-info
# metadata (safe — these are upstream test suites, not runtime code).
find "$PYROOT/lib"/python*/site-packages -maxdepth 2 -type d -name "tests" -path "*/site-packages/*" -prune -exec rm -rf {} + 2>/dev/null || true

# Detect installed versions for metadata. Each lookup is tolerant: an empty
# result must NOT abort the script (set -e + pipefail), so we guard with `|| true`.
ANSIBLE_REPORTED="$("$PY" -m pip show ansible 2>/dev/null | awk -F': ' '/^Version/{print $2}' || true)"
CORE_REPORTED="$("$PY" -m pip show ansible-core 2>/dev/null | awk -F': ' '/^Version/{print $2}' || true)"

# Read collection version from MANIFEST.json (collection_info.version).
# Source trees without a manifest fall back to "local".
COLLECTION_MANIFEST="$YDB_DEST/MANIFEST.json"
COLLECTION_REPORTED="$("$PY" - "$COLLECTION_MANIFEST" <<'PYEOF' 2>/dev/null || true
import json, sys
try:
    with open(sys.argv[1]) as f:
        data = json.load(f)
    print(data.get("collection_info", {}).get("version", ""))
except Exception:
    pass
PYEOF
)"
if [[ -z "$COLLECTION_REPORTED" ]]; then
  COLLECTION_REPORTED="local"
fi

# Same for the EM collection — read the version straight from its MANIFEST.json.
EM_MANIFEST="$EM_DEST/MANIFEST.json"
EM_COLLECTION_REPORTED="$("$PY" - "$EM_MANIFEST" <<'PYEOF' 2>/dev/null || true
import json, sys
try:
    with open(sys.argv[1]) as f:
        data = json.load(f)
    print(data.get("collection_info", {}).get("version", ""))
except Exception:
    pass
PYEOF
)"
if [[ -z "$EM_COLLECTION_REPORTED" ]]; then
  # A local path install may skip MANIFEST.json; fall back to galaxy.yml.
  # Pure-bash extraction avoids quoting pitfalls in heredocs on old bash.
  if [[ -f "$EM_DEST/galaxy.yml" ]]; then
    _ver=$(awk -F':?' '/^version:/{gsub(/[[:space:]""'"'"']/,"",$2); print $2; exit}' "$EM_DEST/galaxy.yml" 2>/dev/null || true)
    [[ -n "$_ver" ]] && EM_COLLECTION_REPORTED="$_ver"
    unset _ver
  fi
fi
if [[ -z "$EM_COLLECTION_REPORTED" ]]; then
  EM_COLLECTION_REPORTED="${EM_COLLECTION_GIT_REF:-unknown}"
fi

echo "==> Packing archive: $ARCHIVE"
# Pack with stable, relative paths: python/ and collections/ at the root.
rm -f "$ARCHIVE"
tar -czf "$ARCHIVE" -C "$WORK" python collections

SHA="$(sha256sum "$ARCHIVE" | awk '{print $1}')"

echo "==> Writing metadata: $VERSION_FILE"
cat > "$VERSION_FILE" <<JSON
{
  "embedded": true,
  "python": "${CPYTHON_VERSION}",
  "ansible": "${ANSIBLE_REPORTED:-$ANSIBLE_VERSION}",
  "ansible_core": "${CORE_REPORTED:-$ANSIBLE_CORE_VERSION}",
  "collection": "${COLLECTION_REPORTED:-unknown}",
  "em_collection": "${EM_COLLECTION_REPORTED:-unknown}",
  "sha256": "${SHA}",
  "collections_subdir": "collections",
  "root_subdir": "python"
}
JSON

echo "==> Caching runtime artefacts: $ARTEFACTS_DIR"
mkdir -p "$ARTEFACTS_DIR"
cp "$ARCHIVE" "$ARTEFACT_ARCHIVE"
cp "$VERSION_FILE" "$ARTEFACT_VERSION"

echo "==> Done."
echo "    archive : $ARCHIVE ($(du -h "$ARCHIVE" | awk '{print $1}'))"
echo "    sha256  : $SHA"
echo "    python  : ${CPYTHON_VERSION}"
echo "    ansible : ${ANSIBLE_REPORTED:-$ANSIBLE_VERSION}"
echo "    collection   (ydb)   : ${COLLECTION_REPORTED:-unknown}"
echo "    collection (ydb_em)  : ${EM_COLLECTION_REPORTED:-unknown}"
