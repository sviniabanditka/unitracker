# Baby Tracker — документация проекта

Self-hosted веб-приложение для трекинга потребностей младенца.

## Зачем

Семейная установка на собственный сервер/NAS/Raspberry Pi. Хранит произвольно настраиваемые трекеры (питание, сон, лекарства, подгузники, …), поддерживает несколько детей, ведёт версионирование и автоматические бекапы с возможностью отката.

## Стек

- **Backend**: Go, `chi`, SQLite (`modernc.org/sqlite`, без CGO), миграции `pressly/goose`, планировщик `gocron`, JSON-Schema валидация (`santhosh-tekuri/jsonschema/v6`).
- **Frontend**: Vue 3 + TypeScript + Vite, Pinia, Vue Router, Tailwind CSS, shadcn-vue, vue-i18n, vite-plugin-pwa, vee-validate + zod.
- **Deploy**: один Docker-образ (multi-stage), фронт встраивается в Go-бинарник через `embed`, volume `/data`.

## Архитектурные решения (общие)

- **Один контейнер** — backend серверит API и встроенный SPA-билд. Никаких отдельных nginx/сервисов.
- **SQLite** — один файл = бекап. WAL-режим. `VACUUM INTO` для атомарного снапшота.
- **Сессии**, не JWT. Хранятся в БД, secure httpOnly cookie. bcrypt для паролей.
- **Регистрация только из админки**. Первый админ — bootstrap из env при пустой `users`.
- **Кастомные трекеры**: `trackers.schema_json` описывает поля, валидируется JSON-Schema. Entries хранят `data_json`. Один из полей может быть помечен `isPrimaryTime: true` — его значение копируется в системный столбец `entries.occurred_at` для сортировки и графиков.
- **Версионирование двух уровней**:
  - **Snapshot** — атомарная копия всей БД на диск. Авто по расписанию (интервал из settings, дефолт 6 часов) + по запросу. Restore — через maintenance-gate без рестарта контейнера, с авто `pre-restore` снапшотом.
  - **Entry revision** — лог изменений каждой записи; в UI можно откатить отдельную запись без отката всей системы.
- **Mobile-first PWA**: install-to-home-screen, offline shell.
- **i18n**: EN + UK, переключатель в шапке. Labels полей трекеров тоже хранятся локализованными.

## Модель данных

```
users(id, username UNIQUE, password_hash, role['admin'|'user'], created_at)
sessions(id PK, user_id FK, expires_at, created_at)
children(id, name, birthday, avatar_url, created_at)
trackers(id, name, icon, color, description, schema_json, is_archived, created_at, updated_at)
entries(id, tracker_id FK, child_id FK NULL, data_json, occurred_at, created_by FK,
        is_deleted, created_at, updated_at)
entry_revisions(id, entry_id FK, data_json, occurred_at, child_id, is_deleted,
                change_type['create'|'update'|'delete'|'restore'], changed_by FK, changed_at)
snapshots(id, filename UNIQUE, size_bytes, type['auto'|'manual'|'pre-restore'],
          note, created_by FK, created_at)
settings(key PK, value, updated_at)
```

Дефолтные `settings`: `backup_interval_hours=6`, `backup_retention_count=20`, `app_name=Baby Tracker`, `default_locale=en`.

## Поддерживаемые типы полей трекера

`datetime`, `date`, `time`, `duration`, `number` (с `unit`/`min`/`max`), `text`, `longtext`, `select`, `multiselect`, `boolean`, `color`.

Пример `schema_json`:

```json
{
  "fields": [
    {"key":"occurred_at","label":{"en":"When","uk":"Коли"},"type":"datetime","required":true,"isPrimaryTime":true},
    {"key":"amount","label":{"en":"Amount","uk":"Кількість"},"type":"number","unit":"ml","min":0,"required":true},
    {"key":"note","label":{"en":"Note","uk":"Примітка"},"type":"longtext"}
  ]
}
```

## Фазы реализации

Идём строго по фазам, каждая — отдельная итерация.

| #  | Файл                                                       | Цель                                            | Статус |
|----|------------------------------------------------------------|-------------------------------------------------|--------|
| 1  | [phase-1-foundation.md](./phase-1-foundation.md)           | Скелет монорепо, Docker, dev-окружение          | [x]    |
| 2  | [phase-2-auth.md](./phase-2-auth.md)                       | Сессии, login, роли, админ-CRUD пользователей    | [x]    |
| 3  | [phase-3-children-trackers.md](./phase-3-children-trackers.md) | Дети, трекеры, SchemaEditor                  | [x]    |
| 4  | [phase-4-entries.md](./phase-4-entries.md)                 | Динамические формы, CRUD записей                | [x]    |
| 5  | [phase-5-revisions.md](./phase-5-revisions.md)             | История ревизий записи + откат                  | [x]    |
| 6  | [phase-6-backup.md](./phase-6-backup.md)                   | Снапшоты БД, scheduler, restore                 | [x]    |
| 7  | [phase-7-admin-settings.md](./phase-7-admin-settings.md)   | Страница настроек, горячее обновление gocron    | [x]    |
| 8  | [phase-8-i18n-pwa.md](./phase-8-i18n-pwa.md)               | i18n EN/UK, PWA, mobile polish, dashboard       | [x]    |
| 9  | [phase-9-charts.md](./phase-9-charts.md)                   | Интерактивные графики на странице трекера       | [x]    |
| 10 | [phase-10-rebrand-profiles-ipad.md](./phase-10-rebrand-profiles-ipad.md) | Profiles + clone-from-library + per-user access + iPad-tier | [x]    |

## Структура репозитория (целевая)

```
baby-tracker/
├── backend/
│   ├── cmd/server/main.go
│   ├── internal/{auth,users,children,trackers,entries,backup,settings,db,http}/
│   ├── migrations/
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/{api,components,views,stores,router,i18n,lib}/
│   ├── public/
│   ├── index.html
│   ├── vite.config.ts
│   ├── tailwind.config.js
│   └── package.json
├── deploy/
│   ├── Dockerfile
│   └── docker-compose.yml
├── docs/                ← вы здесь
├── .env.example
└── README.md
```

## Env переменные

| Имя | Обязательность | Назначение |
|---|---|---|
| `DATA_DIR` | опц., default `/data` | Корень для `app.db` и `backups/` |
| `HTTP_ADDR` | опц., default `:8080` | Адрес HTTP-сервера |
| `SESSION_SECRET` | обязателен | Ключ подписи сессионных cookies (32+ байта) |
| `INITIAL_ADMIN_USERNAME` | при пустой users | Bootstrap админ |
| `INITIAL_ADMIN_PASSWORD` | при пустой users | Bootstrap админ |
| `TZ` | опц., default UTC | Часовой пояс контейнера |

## Запуск

**Production:**
```bash
docker compose -f deploy/docker-compose.yml up --build
# → http://localhost:8080
```

**Dev (после Phase 1):**
```bash
# Терминал 1
cd backend && go run ./cmd/server
# Терминал 2
cd frontend && npm run dev   # vite на :5173, проксирует /api → :8080
```
