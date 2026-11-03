# Архитектура проекта YDB Installer — STAGE II

## Ссылки на документацию по экранам

| Файл | Описание |
|------|----------|
| [`STAGE-II-VIEW.md`](STAGE-II-VIEW.md) | Подробное описание экранов |
| [`STAGE-II-VIEW-ASCII.md`](STAGE-II-VIEW-ASCII.md) | Визуальные схемы всех экранов |

---

## Обзор

YDB Installer — локальный веб-инсталлятор для развёртывания кластеров YDB и YDB EM через Ansible.
Пользователь запускает единый бинарный файл на машине-оркестраторе, открывает браузер и проходит
пошаговый мастер установки.

**Стек:**
- **Backend:** Go (Golang) — HTTP/WebSocket API-сервер, бизнес-логика, запуск Ansible
- **Frontend:** Vue.js 3 + Vite + Pinia + Vue Router + Vuetify 3 — SPA, встроенный в бинарник через `embed.FS`
- **UI-библиотека:** Vuetify 3 — Material Design компоненты с кастомной темой YDB
- **Протокол реального времени:** WebSocket (нативный) — стриминг логов Ansible
- **Установка:** Ansible + коллекция `ydb_platform.ydb`

---

## 1. Структура проекта

```
ydb-installer/
├── cmd/installer/
│   └── main.go                   # Точка входа — единый бинарник
├── internal/
│   ├── api/                      # HTTP/WebSocket API (handlers, middleware, router)
│   ├── services/                 # Бизнес-логика (checker, config, tls, ansible, versions, session)
│   ├── models/                   # Структуры данных (inventory, config, session, task, host)
│   └── embed/                    # Встраивание Vue SPA в бинарник
├── ui/                           # Vue.js 3 SPA (исходники)
│   ├── src/
│   │   ├── plugins/              # Vuetify — инициализация, тема, иконки
│   │   │   └── vuetify.ts        # Настройка темы YDB, подключение компонентов
│   │   ├── router/               # Vue Router — маршруты по экранам
│   │   ├── stores/               # Pinia stores (session, topology, hosts, disks, etc.)
│   │   ├── views/                # Экраны (Screen01.vue ... Screen33.vue)
│   │   ├── components/           # Переиспользуемые компоненты (layout, forms, common)
│   │   ├── composables/          # useWebSocket, useValidation, useAutoFill
│   │   ├── api/                  # HTTP-клиент
│   │   ├── locales/              # ru.json, en.json
│   │   └── types/                # TypeScript типы
│   ├── public/
│   ├── index.html
│   ├── vite.config.ts            # Proxy /api/* и /ws/* → Go в dev-режиме
│   ├── tsconfig.json
│   └── package.json
├── test/
│   ├── unit/                     # Unit-тесты (checker, config, models, session, ansible, api)
├── scripts/
│   ├── build.sh                  # vite build + go build
│   └── dev.sh                    # air + vite dev
├── docs/
├── inventory/                    # Ansible inventory файлы
│   ├── 50-inventory.yaml         # Основной inventory (генерируется инсталлятором)
│   └── 99-inventory-vault.yaml   # Зашифрованные пароли (ansible-vault)
├── files/                        # Конфигурации и локальные дистрибутивы
│   └── config.yaml               # Конфигурация кластера YDB
├── ansible.cfg                   # Конфигурация Ansible
├── ansible_vault_password_file   # Пароль для ansible-vault (plaintext)
├── go.mod
├── go.sum
├── Dockerfile
└── README.md
```

---

## 2. Архитектура Backend (Go)

### 2.1 Точка входа

`cmd/installer/main.go` выполняет:
1. Парсинг флагов:
   - `--port PORT` — порт HTTP-сервера (по умолчанию 8080)
   - `--host HOST` — bind address для HTTP-сервера (по умолчанию `127.0.0.1`; для внешнего доступа укажите `0.0.0.0`)
   - `--no-browser` — не открывать браузер автоматически
   - `--clean-state` — очистить состояние сессии (удалить `inventory.yaml`, `config.yaml`, `TLS/CA/`)
2. Автоопределение языка системы:
   - Проверка переменных окружения: `LANG`, `LC_ALL`, `LANGUAGE`
   - Если язык содержит `ru` или `RU` → русский (`ru`)
   - Иначе → английский (`en`)
   - Язык по умолчанию сохраняется в `Session.DefaultLang`
3. Инициализацию сервисов (session store, config watcher)
4. Регистрацию маршрутов через `internal/api/router.go`
5. Встраивание собранного Vue SPA через `embed.FS`
6. Запуск HTTP-сервера и открытие браузера (если не `--no-browser`)

### 2.2 Слои Backend

```
┌─────────────────────────────────────────────────────┐
│                   HTTP / WebSocket                   │
├─────────────────────────────────────────────────────┤
│  Handlers  (internal/api/handlers/)                 │
│  Маршрутизация, десериализация запросов,            │
│  сериализация ответов, HTTP-коды ошибок             │
├─────────────────────────────────────────────────────┤
│  Services  (internal/services/)                     │
│  Бизнес-логика: SSH-проверки, генерация конфигов,   │
│  запуск Ansible, генерация TLS, получение версий    │
├─────────────────────────────────────────────────────┤
│  Models  (internal/models/)                         │
│  Структуры данных: inventory, config, session,      │
│  task, host                                         │
└─────────────────────────────────────────────────────┘
```

### 2.3 Ключевые сервисы

| Сервис | Файл | Ответственность |
|--------|------|-----------------|
| `DepsChecker` | `services/checker/deps.go` | Проверка Python, Ansible, ansible-коллекции |
| `DepsInstaller` | `services/checker/installer.go` | Автоматическая установка зависимостей через sudo |
| `DepsInstructionProvider` | `services/checker/instruction.go` | Генерация инструкций установки для разных ОС |
| `SSHChecker` | `services/checker/ssh.go` | SSH-доступность, время, Python, диски на хостах |
| `TLSChecker` | `services/checker/tls.go` | Валидация готовых сертификатов |
| `InventoryService` | `services/config/inventory.go` | Парсинг и автоматическое обновление `inventory.yaml` при изменении полей в UI |
| `YDBConfigService` | `services/config/ydbconfig.go` | Парсинг/генерация `config.yaml` |
| `ConfigWatcher` | `services/config/watcher.go` | Поиск конфигов в CWD при старте, генерация шаблонных файлов при отсутствии |
| `TLSGenerator` | `services/tls/generator.go` | Генерация сертификатов через `TLS/ydb-ca-update.sh` |
| `AnsibleRunner` | `services/ansible/runner.go` | Запуск `ansible-playbook` из системного PATH, стриминг stdout |
| `AnsibleCancel` | `services/ansible/cancel.go` | Отмена запущенного процесса (отправка `SIGINT` процессу `ansible-playbook`) |
| `AnsibleParser` | `services/ansible/parser.go` | Парсинг вывода Ansible → задачи + статусы |
| `VersionsFetcher` | `services/versions/fetcher.go` | HTTP-запрос к репозиторию YDB/YDB EM |
| `SessionStore` | `services/session/store.go` | In-memory хранилище состояния текущей установки (с `sync.RWMutex`) |
| `FileLogger` | `services/logger/file.go` | Запись логов Ansible и ошибок Go в файлы `/tmp/ydb-installer/logs/` |

**Примечание по Ansible:** Инсталлятор использует `ansible-playbook` из системного окружения (PATH) и коллекцию `ydb_platform.ydb`, установленную в системе. Playbooks не встраиваются в бинарник — они поставляются вместе с коллекцией.

### 2.4 Session Store

Хранит единственную активную сессию (один пользователь) **только в памяти** Go-сервера. При обновлении страницы фронтенд делает `GET /api/session` и восстанавливает UI из ответа. Для защиты от race conditions используется `sync.RWMutex`.

---

## 3. API Contract

### 3.1 REST Endpoints

#### Система
| Метод | Путь | Описание |
|-------|------|----------|
| `GET` | `/api/system/deps` | Проверка Python, Ansible, ansible-коллекции |
| `GET` | `/api/system/deps/instruction/:os` | Получение инструкции установки для указанной ОС |
| `POST` | `/api/system/deps/install` | Запуск автоматической установки зависимостей через sudo |
| `GET` | `/api/system/configs` | Поиск и парсинг `inventory.yaml` / `config.yaml` в CWD |

#### Хосты
| Метод | Путь | Описание |
|-------|------|----------|
| `POST` | `/api/hosts/check` | SSH-проверка списка хостов |

#### Диски
| Метод | Путь | Описание |
|-------|------|----------|
| `POST` | `/api/disks/scan` | Сканирование дисков на хостах через Ansible |

#### Конфигурация
| Метод | Путь | Описание |
|-------|------|----------|
| `POST` | `/api/config/inventory/validate` | Валидация существующего `inventory.yaml` |
| `POST` | `/api/config/ydb/generate` | Генерация `config.yaml` |
| `POST` | `/api/config/ydb/validate` | Валидация `config.yaml` относительно сессии |

#### TLS
| Метод | Путь | Описание |
|-------|------|----------|
| `POST` | `/api/tls/generate` | Генерация самоподписанных сертификатов |
| `POST` | `/api/tls/validate` | Проверка готовых сертификатов |

#### Версии
| Метод | Путь | Описание |
|-------|------|----------|
| `GET` | `/api/versions/ydb` | Список версий YDB из репозитория |
| `GET` | `/api/versions/ydb-em` | Список версий YDB EM из репозитория |

#### Установка / Удаление
| Метод | Путь | Описание |
|-------|------|----------|
| `POST` | `/api/install/ydb/start` | Запуск установки YDB |
| `POST` | `/api/install/ydb-em/start` | Запуск установки YDB EM |
| `POST` | `/api/install/delete-ydb/start` | Запуск удаления YDB |
| `POST` | `/api/install/delete-em-ydb/start` | Запуск удаления EM+YDB |
| `POST` | `/api/install/cancel` | Отмена текущей операции |
| `GET` | `/api/install/status` | Текущий статус (задачи, прогресс) |

#### Сессия
| Метод | Путь | Описание |
|-------|------|----------|
| `GET` | `/api/session` | Получить текущее состояние сессии (включая `default_lang`) |
| `PUT` | `/api/session` | Обновить состояние сессии (шаг, данные) |

### 3.2 WebSocket

| Путь | Описание |
|------|----------|
| `ws://localhost:PORT/ws/logs` | Стриминг логов Ansible построчно |

---

## 4. Архитектура Frontend (Vue.js 3)

### 4.1 Pinia Stores

| Store | Файл | Данные |
|-------|------|--------|
| `useSessionStore` | `stores/session.ts` | Режим, текущий шаг, статусы шагов для sidebar, `default_lang` (из системы) |
| `useTopologyStore` | `stores/topology.ts` | Выбранная топология, per-topology state (сохраняется при переключении) |
| `useHostsStore` | `stores/hosts.ts` | Хосты по DC, SSH-настройки, результаты проверки |
| `useDisksStore` | `stores/disks.ts` | Список дисков, выбранные диски |
| `useDistributionStore` | `stores/distribution.ts` | Источник, версия, пути к локальным файлам |
| `useYdbConfigStore` | `stores/ydbConfig.ts` | ydb_domain, ydb_password, cores_static и др. |
| `useDatabasesStore` | `stores/databases.ts` | Вариант конфигурации, список БД |
| `useYdbEmStore` | `stores/ydbEm.ts` | Хосты EM, параметры, дистрибутив, TLS |
| `useTlsStore` | `stores/tls.ts` | Режим TLS, пути к сертификатам, статус генерации |
| `useInstallStore` | `stores/install.ts` | Задачи, прогресс, логи, статус WebSocket |

### 4.2 Composables

| Composable | Файл | Назначение |
|------------|------|------------|
| `useWebSocket` | `composables/useWebSocket.ts` | Подключение к `/ws/logs`, обработка сообщений |
| `useValidation` | `composables/useValidation.ts` | Валидация текущего шага перед переходом |
| `useAutoFill` | `composables/useAutoFill.ts` | Автозаполнение stores из данных конфигов |

### 4.3 Переиспользуемые компоненты

| Компонент | Файл | Используется на экранах | Vuetify-основа |
|-----------|------|------------------------|----------------|
| `SshSettings` | `components/forms/SshSettings.vue` | 3, 20, 30, 32 | `v-text-field`, `v-radio-group`, `v-checkbox` |
| `HostTable` | `components/forms/HostTable.vue` | 3, 7, 7.1, 20 | `v-data-table` |
| `TaskList` | `components/common/TaskList.vue` | 9, 24, 31, 33 | `v-list`, `v-list-item` |
| `ConsoleOutput` | `components/common/ConsoleOutput.vue` | 9, 24, 31, 33 | `v-virtual-scroll`, `v-card` |
| `ProgressBar` | `components/common/ProgressBar.vue` | 9, 24, 31, 33 | `v-progress-linear` |
| `ConfirmDialog` | `components/common/ConfirmDialog.vue` | 9, 24, 31, 33 | `v-dialog` |
| `AppSidebar` | `components/layout/AppSidebar.vue` | Все экраны | `v-navigation-drawer`, `v-list` |

### 4.4 Инициализация языка

При загрузке приложения:
1. `GET /api/session` → получает `default_lang` (автоопределённый на Backend)
2. `useSessionStore` сохраняет `default_lang`
3. Vue I18n устанавливает `locale` = `default_lang`
4. Пользователь может переключить язык через переключатель в шапке → сохраняется в `localStorage`

### 4.5 Vite конфигурация

В dev-режиме Vite проксирует `/api/*` и `/ws/*` на Go-сервер (`:8080`).
В production `ui/dist/` встраивается в Go-бинарник через `embed.FS` и раздаётся напрямую.

### 4.6 Vuetify 3

**Версия:** Vuetify 3 (совместима с Vue 3 + Composition API).

**Роль:** Полный Material Design UI-фреймворк с кастомной темой YDB. Предоставляет готовые компоненты для всех 33 экранов мастера установки — формы, таблицы, диалоги, прогресс-бары, навигацию.

**Подключение:** `ui/src/plugins/vuetify.ts` — инициализация плагина, настройка темы, регистрация иконок (MDI).

**Tree-shaking:** Через `vite-plugin-vuetify` с опцией `autoImport: true` — в финальный бандл попадают только используемые компоненты. Это критично для встраивания SPA в Go-бинарник через `embed.FS`.

**Кастомная тема YDB:**
- Переопределение цветовых токенов (`primary`, `secondary`, `error`, `success`) под брендинг YDB
- Настройка через `createVuetify({ theme: { themes: { light: { colors: { ... } } } } })`
- Поддержка light/dark режимов

**Ключевые компоненты по категориям:**

| Категория | Компоненты Vuetify | Применение в проекте |
|-----------|-------------------|----------------------|
| Layout | `v-app`, `v-main`, `v-navigation-drawer`, `v-app-bar`, `v-footer` | Общий лейаут всех экранов, `AppSidebar` |
| Формы | `v-text-field`, `v-select`, `v-checkbox`, `v-radio-group`, `v-slider`, `v-switch` | `SshSettings`, экраны 3, 6, 7, 20, 30 |
| Таблицы | `v-data-table` | `HostTable` — экраны 3, 7, 7.1, 20 |
| Диалоги | `v-dialog` | `ConfirmDialog` — экраны 9, 24, 31, 33 |
| Прогресс | `v-progress-linear` | `ProgressBar` — экраны 9, 24, 31, 33 |
| Списки | `v-list`, `v-list-item` | `TaskList`, sidebar-шаги |
| Прокрутка | `v-virtual-scroll` | `ConsoleOutput` — вывод логов Ansible |
| Карточки | `v-card`, `v-card-title`, `v-card-text` | Блоки контента на всех экранах |
| Кнопки | `v-btn`, `v-btn-group` | Навигация «Назад» / «Далее», действия |
| Уведомления | `v-alert`, `v-chip` | Статусы зависимостей, предупреждения |
| Выпадающие | `v-expansion-panels` | «Расширенные настройки» на экранах 3, 6, 7 |

---

## 5. Механизм генерации TLS-сертификатов

Инсталлятор использует существующий скрипт [`TLS/ydb-ca-update.sh`](TLS/ydb-ca-update.sh) для генерации самоподписанных сертификатов.

**API-эндпоинты:**
- `POST /api/tls/generate` — генерация сертификатов для списка хостов
- `POST /api/tls/validate` — валидация готовых сертификатов

**Архитектурное решение:**
- Сервис `TLSGenerator` формирует файл нод и вызывает скрипт генерации
- Сертификаты сохраняются в директорию `TLS/CA/certs/certs-per-host/`
- Путь к сертификатам сохраняется в сессии и добавляется в `inventory.yaml`
- Сервис `TLSChecker` проверяет наличие и валидность сертификатов для каждого хоста

---

## 6. Проверка и установка зависимостей

Инсталлятор предоставляет три варианта действий при отсутствии зависимостей:

1. **Получить инструкцию** - API `GET /api/system/deps/instruction/:os` возвращает команды установки для текущей ОС (Ubuntu/Debian, CentOS/RHEL, macOS)
2. **Установить автоматически** - API `POST /api/system/deps/install` запускает установку через sudo с стримингом лога через WebSocket
3. **Пропустить** - продолжить без установки с предупреждением о возможных проблемах

**Архитектурное решение:**
- Сервис `DepsInstaller` выполняет установку через `exec.Command` с `sudo`
- Вывод команд стримится в реальном времени
- Требуется подтверждение пользователя перед запуском
- При ошибке - отображается сообщение с рекомендациями

---

## 7. Потоки данных

### 7.1 Автозаполнение при старте

```
Запуск бинарника
  → Автоопределение языка системы (LANG, LC_ALL, LANGUAGE)
      → Если содержит ru/RU → default_lang = "ru"
      → Иначе → default_lang = "en"
  → ConfigWatcher ищет inventory.yaml / config.yaml в CWD
      → Если файлы найдены → парсит и заполняет Session
      → Если файлы отсутствуют → генерирует шаблонные версии с минимальными значениями
  → GET /api/system/configs
  → Frontend получает данные
  → useAutoFill composable → заполняет все Pinia stores
  → Vue I18n устанавливает locale = default_lang
```

---

## 8. Логирование

Инсталлятор использует сервис `FileLogger` для записи всех логов в файлы на диске для диагностики проблем после завершения установки.

**Путь к логам:** `/tmp/ydb-installer/logs/`

**Формат имени файла:** `install-YYYY-MM-DD.log` (например, `install-2024-01-15.log`)

**Что записывается:**
- Весь stdout/stderr от `ansible-playbook` (построчно)
- Внутренние ошибки Go-сервера
- Статусы задач и прогресс

**Интеграция с API:**
- Путь к текущему лог-файлу возвращается в ответе `GET /api/install/status`
- Отображается на Экранах 10, 25, 31, 33 для пользователя
- Кнопка «Скопировать путь» позволяет быстро получить путь к логу

---

## 9. Интеграция с Ansible

Инсталлятор использует Ansible для развертывания YDB и YDB EM на целевых хостах. Инсталлятор генерирует и управляет следующими файлами Ansible-окружения:

**Структура файлов:**
- `inventory/50-inventory.yaml` — основной inventory файл (генерируется инсталлятором)
- `inventory/99-inventory-vault.yaml` — зашифрованные пароли (ansible-vault)
- `files/config.yaml` — конфигурация кластера YDB
- `ansible.cfg` — конфигурация Ansible
- `ansible_vault_password_file` — пароль для ansible-vault

**Архитектурное решение:**
- При запуске инсталлятора сервис `ConfigWatcher` проверяет наличие `inventory.yaml` и `config.yaml` в CWD
- Если файлы отсутствуют - генерируются шаблонные версии с минимальными значениями
- Сервис `InventoryService` автоматически обновляет `inventory.yaml` при каждом изменении полей в UI
- Сервис `YDBConfigService` генерирует `config.yaml` по кнопке на Экране 9 или 24
- Сервис `AnsibleRunner` запускает `ansible-playbook` из системного PATH
- Логи стримятся через WebSocket (`/ws/logs`)
- При следующем запуске инсталлятора эти файлы используются для автозаполнения

**Примечание:** Инсталлятор использует `ansible-playbook` из системного окружения (PATH) и коллекцию `ydb_platform.ydb`, установленную в системе. Playbooks не встраиваются в бинарник — они поставляются вместе с коллекцией.

---

## 10. Тестирование

Для STAGE II используется только **unit-тестирование**.

**Выбранные инструменты:**
- **Backend (Go):** стандартный пакет `testing` + `testify` для assertions
- **Frontend (Vue.js):** Vitest (рекомендуется для Vite)
- **Запуск тестов:** через Makefile (без CI/CD)

**Структура тестов:**
- Backend: `test/unit/` — тесты для checker, config, models, session, ansible, api handlers
- Frontend: тесты располагаются рядом с тестируемыми файлами (stores, composables, components)
