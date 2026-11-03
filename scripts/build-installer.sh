#!/usr/bin/env bash
# Собрать инсталлятор, встроив коллекцию ydb_platform.ydb из этого репозитория.
#
# Скрипт лежит в корне площадки, вне installer/ и вне ansible-collection/,
# чтобы эти каталоги можно было отдавать обратно в исходные репозитории
# без интеграционной обвязки.
#
# installer/scripts/build-runtime.sh сначала ставит коллекцию из
# COLLECTION_GIT_URL (installer/scripts/runtime.lock), и переменная окружения
# эту строку не перебивает: lock-файл присваивает её заново. Локальный путь
# используется только если URL пуст. Поэтому на время сборки скрипт очищает
# COLLECTION_GIT_URL в lock-файле и передаёт COLLECTION_LOCAL_PATH на
# ansible-collection/. Исходные байты lock-файла лежат в .build-installer/
# и возвращаются при выходе, в том числе после ошибки. Постоянной ссылки
# на эту площадку в lock-файле не появляется.
#
# Кэш installer/artefacts/ не запоминает, какое дерево коллекции упаковано.
# По умолчанию кэш обходится (FORCE_RUNTIME_BUILD=1), иначе в бинарник
# может попасть ранее скачанный git-снимок, а не плейбуки из соседнего каталога.
#
# Если процесс убить через SIGKILL, lock-файл останется очищенным, а копия
# исходника — в .build-installer/runtime.lock.orig. Следующий запуск
# восстановит файл сам. Вручную: cp -p .build-installer/runtime.lock.orig
# installer/scripts/runtime.lock
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
INSTALLER="$REPO/installer"
COLLECTION="$REPO/ansible-collection"
LOCK_FILE="$INSTALLER/scripts/runtime.lock"
STATE_DIR="$REPO/.build-installer"
LOCK_BACKUP="$STATE_DIR/runtime.lock.orig"
OUT_DIR="$REPO/dist"

# Плейбуки, которые инсталлятор вызывает по FQCN ydb_platform.ydb.<name>.
REQUIRED_PLAYBOOKS=(initial_setup install_dynamic format_cluster)

SKIP_RUNTIME=0
RUNTIME_ONLY=0
ALLOW_CACHE=0
PRINT_PLAN=0
SELF_TEST=0
# Глобальная: EXIT-ловушка срабатывает уже после выхода из main,
# и local-переменная к этому моменту уничтожена.
runtime_started=0

cleanup() {
  local status=$?
  if ! restore_lock_file; then
    status=1
  fi
  if [[ "$runtime_started" -eq 1 && "$status" -ne 0 ]]; then
    # Не оставлять многомегабайтный архив в дереве installer/ после сбоя.
    git -C "$REPO" checkout HEAD -- \
      installer/internal/runtime/assets/runtime-linux-amd64.tar.gz \
      installer/internal/runtime/assets/runtime.version >/dev/null 2>&1 || true
  fi
  trap - EXIT
  exit "$status"
}

usage() {
  cat <<'EOF'
Собрать инсталлятор с плейбуками из ansible-collection/ этого репозитория.

  scripts/build-installer.sh [опции]

Опции:
  --runtime-only   Собрать только встроенный Python+Ansible runtime
  --skip-runtime   Не пересобирать runtime (UI и бинарник Go)
  --allow-cache    Взять архив из installer/artefacts/, если он уже есть.
                   В кэше может лежать коллекция из git, а не из этого каталога
  --out DIR        Куда положить результат (по умолчанию: <корень>/dist)
  --print-plan     Показать параметры сборки и выйти, ничего не меняя
  --self-test      Проверить обвязку на копии lock-файла, сборку не запускать
  -h, --help

Результат полной сборки: dist/ydb-installer
(копия также остаётся в installer/ydb-installer, как пишет штатный build.sh).

Результат --runtime-only: dist/runtime-linux-amd64.tar.gz и dist/runtime.version.
Плейсхолдер в installer/internal/runtime/assets/ после этого возвращается.

Коллекция ydb_platform.ydb берётся из ./ansible-collection. Ссылка
COLLECTION_GIT_URL в installer/scripts/runtime.lock на время сборки
очищается и затем восстанавливается.
EOF
}

die() {
  echo "ERROR: $*" >&2
  exit 1
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --runtime-only) RUNTIME_ONLY=1 ;;
      --skip-runtime) SKIP_RUNTIME=1 ;;
      --allow-cache) ALLOW_CACHE=1 ;;
      --print-plan) PRINT_PLAN=1 ;;
      --self-test) SELF_TEST=1 ;;
      --out)
        [[ $# -ge 2 ]] || die "--out требует каталог"
        OUT_DIR=$2
        shift
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        die "неизвестная опция: $1 (см. --help)"
        ;;
    esac
    shift
  done

  if [[ "$RUNTIME_ONLY" -eq 1 && "$SKIP_RUNTIME" -eq 1 ]]; then
    die "--runtime-only и --skip-runtime вместе не имеют смысла"
  fi
  if [[ "$SELF_TEST" -eq 1 && ( "$RUNTIME_ONLY" -eq 1 || "$SKIP_RUNTIME" -eq 1 || "$PRINT_PLAN" -eq 1 ) ]]; then
    die "--self-test запускается отдельно от сборки"
  fi
}

# Очистить ровно одну строку COLLECTION_GIT_URL=, не трогая похожие имена
# (COLLECTION_GIT_REF, COLLECTION_LOCAL_PATH, EM_COLLECTION_GIT_URL).
blank_collection_git_url() {
  local file=$1
  local tmp line found=0
  tmp="$(mktemp)"
  while IFS= read -r line || [[ -n "$line" ]]; do
    if [[ "$line" == COLLECTION_GIT_URL=* ]]; then
      printf 'COLLECTION_GIT_URL=\n' >> "$tmp"
      found=$((found + 1))
    else
      printf '%s\n' "$line" >> "$tmp"
    fi
  done < "$file"
  if [[ "$found" -ne 1 ]]; then
    rm -f "$tmp"
    echo "ERROR: в $file ожидалась ровно одна строка COLLECTION_GIT_URL=, найдено $found" >&2
    return 1
  fi
  cat "$tmp" > "$file"
  rm -f "$tmp"
}

galaxy_field() {
  local key=$1
  awk -v key="$key" '
    $1 == key":" { print $2; exit }
  ' "$COLLECTION/galaxy.yml"
}

preflight_tree() {
  [[ -f "$COLLECTION/galaxy.yml" ]] || die "нет $COLLECTION/galaxy.yml"
  [[ -f "$LOCK_FILE" ]] || die "нет $LOCK_FILE"
  [[ -f "$INSTALLER/scripts/build.sh" ]] || die "нет installer/scripts/build.sh"
  [[ -f "$INSTALLER/scripts/build-runtime.sh" ]] || die "нет installer/scripts/build-runtime.sh"

  local namespace name
  namespace="$(galaxy_field namespace)"
  name="$(galaxy_field name)"
  [[ "$namespace" == "ydb_platform" && "$name" == "ydb" ]] || \
    die "ansible-collection — это ${namespace:-?}.${name:-?}, ожидается ydb_platform.ydb"

  local pb
  for pb in "${REQUIRED_PLAYBOOKS[@]}"; do
    if [[ ! -f "$COLLECTION/playbooks/$pb.yaml" && ! -f "$COLLECTION/playbooks/$pb.yml" ]]; then
      die "в ansible-collection нет плейбука playbooks/$pb.yaml — инсталлятор вызывает ydb_platform.ydb.$pb"
    fi
  done

  case "$COLLECTION" in
    *[[:space:]]*) die "путь к коллекции содержит пробел: $COLLECTION" ;;
  esac
}

embed_package_present() {
  local found
  found="$(find "$INSTALLER/internal/embed" -name '*.go' -print -quit 2>/dev/null || true)"
  [[ -n "$found" ]]
}

missing_tools() {
  local missing=()
  command -v go >/dev/null 2>&1 || missing+=(go)
  command -v npm >/dev/null 2>&1 || missing+=(npm)
  command -v git >/dev/null 2>&1 || missing+=(git)
  if [[ "$SKIP_RUNTIME" -ne 1 ]]; then
    command -v curl >/dev/null 2>&1 || missing+=(curl)
  fi
  if [[ ${#missing[@]} -gt 0 ]]; then
    printf '%s\n' "${missing[@]}"
  fi
}

resolve_out_dir() {
  if [[ "$OUT_DIR" != /* ]]; then
    OUT_DIR="$PWD/$OUT_DIR"
  fi
}

print_plan() {
  echo "репозиторий:     $REPO"
  echo "инсталлятор:     $INSTALLER"
  echo "коллекция:       $COLLECTION (ydb_platform.ydb)"
  echo "плейбуки:        ${REQUIRED_PLAYBOOKS[*]}"
  echo "lock-файл:       $LOCK_FILE"
  echo "результат:       $OUT_DIR"
  if [[ "$SKIP_RUNTIME" -eq 1 ]]; then
    echo "runtime:         пропуск (SKIP_RUNTIME_BUILD=1)"
  elif [[ "$ALLOW_CACHE" -eq 1 ]]; then
    echo "runtime:         локальная коллекция, кэш installer/artefacts/ разрешён"
  else
    echo "runtime:         локальная коллекция, кэш installer/artefacts/ обходится"
  fi
  if [[ "$RUNTIME_ONLY" -eq 1 ]]; then
    echo "режим:           только runtime"
  else
    echo "режим:           runtime + UI + бинарник"
  fi
  echo "на время сборки: COLLECTION_GIT_URL очищается, затем lock-файл восстанавливается"
  echo "COLLECTION_LOCAL_PATH=$COLLECTION"
  if [[ -f "$LOCK_BACKUP" ]]; then
    echo "внимание:        есть копия прерванной сборки ($LOCK_BACKUP); запуск её восстановит"
  fi
  local tools
  tools="$(missing_tools || true)"
  if [[ -n "$tools" ]]; then
    echo "не найдены:      ${tools//$'\n'/ }"
  fi
  if [[ "$RUNTIME_ONLY" -ne 1 ]] && ! embed_package_present; then
    echo "внимание:        нет пакета installer/internal/embed — go build бинарника не пройдёт"
  fi
}

recover_interrupted_lock() {
  [[ -f "$LOCK_BACKUP" ]] || return 0
  if cmp -s "$LOCK_BACKUP" "$LOCK_FILE"; then
    rm -f "$LOCK_BACKUP"
    return 0
  fi
  echo "==> Восстанавливаю $LOCK_FILE после прерванной сборки"
  cp -p "$LOCK_BACKUP" "$LOCK_FILE"
  if ! cmp -s "$LOCK_BACKUP" "$LOCK_FILE"; then
    die "не удалось восстановить $LOCK_FILE из $LOCK_BACKUP"
  fi
  rm -f "$LOCK_BACKUP"
}

hold_lock_backup() {
  mkdir -p "$STATE_DIR"
  cp -p "$LOCK_FILE" "$LOCK_BACKUP"
  blank_collection_git_url "$LOCK_FILE"
  grep -q '^COLLECTION_GIT_URL=$' "$LOCK_FILE" || die "COLLECTION_GIT_URL не очищен"
  # Непустая строка того же ключа не должна остаться.
  if grep -q '^COLLECTION_GIT_URL=.' "$LOCK_FILE"; then
    die "COLLECTION_GIT_URL остался непустым"
  fi
}

restore_lock_file() {
  [[ -f "$LOCK_BACKUP" ]] || return 0
  cp -p "$LOCK_BACKUP" "$LOCK_FILE"
  if ! cmp -s "$LOCK_BACKUP" "$LOCK_FILE"; then
    echo "ERROR: не удалось вернуть $LOCK_FILE к исходным байтам ($LOCK_BACKUP сохранён)" >&2
    return 1
  fi
  rm -f "$LOCK_BACKUP"
}

# Вернуть плейсхолдер runtime, который build-runtime.sh подменяет большим архивом.
# Нужно, чтобы дерево installer/ не осталось с изменённым встроенным ассетом.
restore_runtime_placeholder() {
  git -C "$REPO" checkout HEAD -- \
    installer/internal/runtime/assets/runtime-linux-amd64.tar.gz \
    installer/internal/runtime/assets/runtime.version
}

publish_binary() {
  [[ -f "$INSTALLER/ydb-installer" ]] || die "сборка не создала $INSTALLER/ydb-installer"
  mkdir -p "$OUT_DIR"
  cp -p "$INSTALLER/ydb-installer" "$OUT_DIR/ydb-installer"
  echo "==> Бинарник: $OUT_DIR/ydb-installer"
}

publish_runtime() {
  local assets="$INSTALLER/internal/runtime/assets"
  [[ -f "$assets/runtime-linux-amd64.tar.gz" ]] || die "runtime-архив не собран"
  [[ -f "$assets/runtime.version" ]] || die "runtime.version не записан"
  mkdir -p "$OUT_DIR"
  cp -p "$assets/runtime-linux-amd64.tar.gz" "$OUT_DIR/runtime-linux-amd64.tar.gz"
  cp -p "$assets/runtime.version" "$OUT_DIR/runtime.version"
  echo "==> Runtime: $OUT_DIR/runtime-linux-amd64.tar.gz"
  restore_runtime_placeholder
}

assert_local_collection_used() {
  local log=$1
  if grep -q "Reusing cached runtime artefacts" "$log"; then
    echo "WARNING: взят кэш installer/artefacts/. Плейбуки из ansible-collection/ могли в него не попасть." >&2
    echo "         Пересоберите без --allow-cache." >&2
    return 0
  fi
  if ! grep -F "ydb source: path:$COLLECTION" "$log" >/dev/null; then
    die "runtime собран не из $COLLECTION (в журнале нет 'ydb source: path:$COLLECTION')"
  fi
}

run_build() {
  local log status
  log="$(mktemp)"
  set +e
  if [[ "$RUNTIME_ONLY" -eq 1 ]]; then
    bash "$INSTALLER/scripts/build-runtime.sh" 2>&1 | tee "$log"
  else
    bash "$INSTALLER/scripts/build.sh" 2>&1 | tee "$log"
  fi
  status=${PIPESTATUS[0]}
  set -e
  if [[ "$status" -ne 0 ]]; then
    rm -f "$log"
    exit "$status"
  fi
  if [[ "$SKIP_RUNTIME" -ne 1 ]]; then
    assert_local_collection_used "$log"
  fi
  rm -f "$log"

  if [[ "$RUNTIME_ONLY" -eq 1 ]]; then
    publish_runtime
  else
    publish_binary
    # build.sh возвращает плейсхолдер только после успешного go build.
    # Повторный checkout безвреден, если плейсхолдер уже на месте, и
    # подчищает архив, если штатное восстановление не выполнилось.
    if [[ "$SKIP_RUNTIME" -ne 1 ]]; then
      restore_runtime_placeholder
    fi
  fi
}

self_test() {
  preflight_tree
  local real_lock real_backup before after work expected
  real_lock="$LOCK_FILE"
  real_backup="$LOCK_BACKUP"
  before="$(sha256sum "$real_lock" | awk '{print $1}')"

  work="$(mktemp)"
  expected="$(mktemp)"
  cp -p "$real_lock" "$work"
  # Строки-ловушки: похожие имена не должны терять значение.
  printf 'EM_COLLECTION_GIT_URL=git+https://example.invalid\n' >> "$work"
  printf 'COLLECTION_GIT_URL_EXTRA=keep\n' >> "$work"
  cp -p "$work" "$expected"

  LOCK_FILE="$work"
  LOCK_BACKUP="$(mktemp)"
  hold_lock_backup

  grep -q '^COLLECTION_GIT_URL=$' "$LOCK_FILE" || die "self-test: URL не очищен"
  if grep -q '^COLLECTION_GIT_URL=.' "$LOCK_FILE"; then
    die "self-test: URL остался непустым"
  fi
  grep -q '^EM_COLLECTION_GIT_URL=git+https://example.invalid$' "$LOCK_FILE" || \
    die "self-test: задета чужая строка EM_COLLECTION_GIT_URL"
  grep -q '^COLLECTION_GIT_URL_EXTRA=keep$' "$LOCK_FILE" || \
    die "self-test: задета строка COLLECTION_GIT_URL_EXTRA"
  grep -q '^COLLECTION_GIT_REF=main$' "$LOCK_FILE" || die "self-test: изменён COLLECTION_GIT_REF"
  grep -q '^COLLECTION_LOCAL_PATH=../ydb-ansible$' "$LOCK_FILE" || \
    die "self-test: изменён COLLECTION_LOCAL_PATH"

  # Повторяет приоритет installer/scripts/build-runtime.sh:
  # env COLLECTION_LOCAL_PATH переживает source, пустой GIT URL пропускается.
  (
    COLLECTION_LOCAL_PATH="$COLLECTION"
    _COLLECTION_OVERRIDE="${COLLECTION_LOCAL_PATH:-}"
    # shellcheck disable=SC1090
    source "$LOCK_FILE"
    [[ -z "${COLLECTION_GIT_URL:-}" ]]
    [[ -n "${_COLLECTION_OVERRIDE}" ]]
    COLLECTION_LOCAL_PATH="$_COLLECTION_OVERRIDE"
    [[ "$COLLECTION_LOCAL_PATH" == "$COLLECTION" ]]
    [[ "${EM_COLLECTION_GIT_URL}" == "git+https://example.invalid" ]]
    [[ "${COLLECTION_GIT_REF}" == "main" ]]
  ) || die "self-test: source lock-файла не отдаёт локальную коллекцию"

  restore_lock_file
  cmp -s "$expected" "$LOCK_FILE" || die "self-test: lock-файл не вернулся к исходным байтам"
  [[ ! -f "$LOCK_BACKUP" ]] || die "self-test: копия lock-файла не удалена"

  # Прерванная сборка: в lock пустой URL, исходные байты лежат в backup.
  local interrupted
  interrupted="$(mktemp)"
  cp -p "$real_lock" "$interrupted"
  LOCK_FILE="$interrupted"
  LOCK_BACKUP="$(mktemp)"
  cp -p "$interrupted" "$LOCK_BACKUP"
  blank_collection_git_url "$LOCK_FILE"
  if cmp -s "$real_lock" "$LOCK_FILE"; then
    die "self-test: очистка не изменила lock-файл"
  fi
  recover_interrupted_lock
  cmp -s "$real_lock" "$LOCK_FILE" || die "self-test: прерванная сборка не восстановила lock"
  [[ ! -f "$LOCK_BACKUP" ]] || die "self-test: backup прерванной сборки не удалён"
  rm -f "$interrupted"

  LOCK_FILE="$real_lock"
  LOCK_BACKUP="$real_backup"
  after="$(sha256sum "$real_lock" | awk '{print $1}')"
  [[ "$before" == "$after" ]] || die "self-test: изменён настоящий $real_lock"
  rm -f "$work" "$expected"
  echo "self-test: ok"
}

main() {
  parse_args "$@"
  resolve_out_dir

  if [[ "$SELF_TEST" -eq 1 ]]; then
    self_test
    exit 0
  fi

  preflight_tree

  if [[ "$PRINT_PLAN" -eq 1 ]]; then
    print_plan
    exit 0
  fi

  local tools
  tools="$(missing_tools || true)"
  if [[ -n "$tools" ]]; then
    die "не найдены команды: ${tools//$'\n'/ }"
  fi
  if [[ "$RUNTIME_ONLY" -ne 1 ]] && ! embed_package_present; then
    die "в копии инсталлятора нет пакета internal/embed (его импортирует cmd/installer). go build из-за этого остановится. Скрипт этот пакет не добавляет: правка была бы внутри installer/."
  fi

  mkdir -p "$STATE_DIR"
  exec 9>"$STATE_DIR/lock"
  if ! flock -n 9; then
    die "другая сборка уже идёт ($STATE_DIR/lock)"
  fi

  recover_interrupted_lock
  runtime_started=0
  trap cleanup EXIT

  if [[ "$SKIP_RUNTIME" -ne 1 ]]; then
    echo "==> Коллекция: $COLLECTION"
    echo "==> На время сборки очищаю COLLECTION_GIT_URL в $LOCK_FILE"
    hold_lock_backup
    export COLLECTION_LOCAL_PATH="$COLLECTION"
    unset SKIP_RUNTIME_BUILD
    runtime_started=1
    if [[ "$ALLOW_CACHE" -eq 1 ]]; then
      export FORCE_RUNTIME_BUILD=0
    else
      export FORCE_RUNTIME_BUILD=1
    fi
  else
    export SKIP_RUNTIME_BUILD=1
  fi

  if [[ -n "${PLATFORM:-}" ]]; then
    export PLATFORM
  fi
  # ansible.cfg инсталлятора пишет лог в ./ydb.log. Уводим его из дерева installer/.
  export ANSIBLE_LOG_PATH="$STATE_DIR/ansible.log"

  run_build
  restore_lock_file
  echo "==> Готово. Lock-файл возвращён к исходным байтам."
}

main "$@"
