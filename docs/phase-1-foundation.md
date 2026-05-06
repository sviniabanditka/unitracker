# Phase 1 — Foundation

## Цель

Развернуть рабочий каркас монорепо: после `docker compose up --build` на `http://localhost:8080` отдаётся placeholder-страница Vue, `GET /api/health` отвечает `{"status":"ok"}`. БД SQLite создаётся в volume, миграции применяются автоматически. Dev-режим тоже работает.

## Объём (Scope)

- Структура директорий монорепо (`backend/`, `frontend/`, `deploy/`).
- Go-модуль с `chi`, `modernc.org/sqlite`, `pressly/goose`, базовая структура `internal/`.
- HTTP-сервер: middleware (logger, recoverer, RequestID), `GET /api/health`, embed SPA-fallback.
- Миграции goose: пустая `0001_init.sql` — пока создаёт таблицу `schema_version` (через goose) и тестовую `_health` (опц.) или просто пустая «up».
- Vue 3 + Vite + TypeScript проект.
- Tailwind CSS, shadcn-vue init (через `npx shadcn-vue@latest init`).
- Pinia + Vue Router установлены, роутер с одной заглушкой `/`.
- Vite dev-server проксирует `/api/*` на `localhost:8080`.
- Multi-stage Dockerfile.
- `docker-compose.yml` с volume `/data`.
- `.env.example` со всеми переменными.

## Не входит в фазу

- Авторизация, login, сессии (Phase 2).
- Реальные таблицы (users, trackers и т.д.) — добавляются по мере появления соответствующих фаз.
- i18n (Phase 8).
- PWA-манифест и service worker (Phase 8).
- Реальные shadcn-vue компоненты в UI (только инициализация и проверочный Button).

## Зависимости

Нет.

## Изменения в БД

- Новая директория `backend/migrations/`.
- `0001_init.sql` — пустая (или ставит `PRAGMA journal_mode=WAL;` через application-level код, не миграцию).
- Goose сам создаст `goose_db_version`.

WAL и `busy_timeout=5000` выставляются в `db.Open()` каждый раз, не в миграциях.

## Backend — что добавляется

```
backend/
├── cmd/server/main.go              # env load, db open+migrate, http server start, graceful shutdown
├── internal/
│   ├── db/
│   │   ├── db.go                   # Open(path) *sql.DB с WAL + busy_timeout
│   │   └── migrate.go              # goose.Up(db, "migrations")
│   └── http/
│       ├── router.go               # chi.Mux, middleware, /api/health
│       ├── spa.go                  # embed.FS для frontend/dist + fallback на index.html
│       └── dist/.gitkeep           # placeholder; реальный билд кладёт сюда Dockerfile
├── migrations/
│   └── 0001_init.sql               # -- +goose Up / -- +goose Down (пустые)
├── go.mod
└── go.sum
```

`main.go` поток:
1. Загрузка env (без сторонних либ — `os.Getenv` достаточно).
2. `DATA_DIR` создаётся если нет.
3. `db.Open(filepath.Join(DATA_DIR, "app.db"))`.
4. `db.Migrate(...)`.
5. `http.NewServer(db)` → `srv.ListenAndServe(addr)`.
6. Сигналы SIGINT/SIGTERM → graceful shutdown с таймаутом.

## Frontend — что добавляется

```
frontend/
├── src/
│   ├── api/client.ts               # fetch wrapper, base URL '/api', credentials: 'include'
│   ├── components/.gitkeep
│   ├── views/Home.vue              # placeholder с одним shadcn-vue Button
│   ├── stores/.gitkeep
│   ├── router/index.ts             # createRouter с одной /  routes
│   ├── App.vue
│   ├── main.ts                     # Vue + Pinia + Router
│   └── style.css                   # tailwind directives
├── public/favicon.svg
├── index.html
├── vite.config.ts                  # proxy /api → :8080
├── tailwind.config.js
├── postcss.config.js
├── tsconfig.json
├── tsconfig.node.json
├── components.json                 # shadcn-vue config
└── package.json
```

`api/client.ts` — единая точка для будущих эндпоинтов; пока умеет только `health()`.

## API эндпоинты

| Метод | Путь | Доступ | Ответ |
|---|---|---|---|
| GET | `/api/health` | публичный | `{"status":"ok","version":"<git-sha>","time":"..."}` |

SPA-fallback: любой не-`/api/*` GET, не нашедший файл в `dist`, отдаёт `index.html`.

## Зависимости (libs)

**Go (`go.mod`):**
- `github.com/go-chi/chi/v5`
- `github.com/go-chi/chi/v5/middleware`
- `modernc.org/sqlite`
- `github.com/pressly/goose/v3`

**JS (`package.json`):**
- `vue@^3`, `vue-router@^4`, `pinia`
- dev: `vite`, `@vitejs/plugin-vue`, `typescript`, `vue-tsc`
- `tailwindcss`, `postcss`, `autoprefixer`
- shadcn-vue зависимости установятся автоматически из `npx shadcn-vue init`: `radix-vue`, `class-variance-authority`, `clsx`, `tailwind-merge`, `lucide-vue-next`, `tailwindcss-animate`.

## Docker

`deploy/Dockerfile` (multi-stage):
1. `node:20-alpine` → `npm ci && npm run build` в `/app/frontend/dist`.
2. `golang:1.22-alpine` → копирует backend, копирует фронт `dist` в `internal/http/dist/`, `go build -trimpath -ldflags="-s -w" -o /server ./cmd/server`.
3. `alpine:3.20` → `apk add ca-certificates tzdata`, бинарник, `EXPOSE 8080`, `VOLUME /data`, `CMD ["/server"]`.

`deploy/docker-compose.yml`:
```yaml
services:
  app:
    build: { context: .., dockerfile: deploy/Dockerfile }
    ports: ["8080:8080"]
    volumes: ["./data:/data"]
    environment:
      - SESSION_SECRET=changeme-32-bytes-or-more-please
    restart: unless-stopped
```

## Acceptance criteria

- [ ] `docker compose -f deploy/docker-compose.yml up --build` поднимает контейнер без ошибок.
- [ ] `curl http://localhost:8080/api/health` → `200 {"status":"ok",...}`.
- [ ] Открытие `http://localhost:8080/` показывает Vue-страницу с одной кнопкой.
- [ ] В `./data/` появился `app.db` и журнальные файлы WAL.
- [ ] Любой неизвестный путь, например `/some/unknown`, всё равно отдаёт `index.html` (SPA-fallback).
- [ ] `cd frontend && npm run dev` поднимает Vite на `:5173`, fetch `/api/health` через прокси работает.
- [ ] Бинарник Go собирается без CGO (`CGO_ENABLED=0`).
- [ ] `.env.example` присутствует и описывает все переменные.

## Verification

1. `cd /Users/bublya/projects/baby-tracker && docker compose -f deploy/docker-compose.yml up --build`.
2. В новом терминале: `curl -s http://localhost:8080/api/health | jq` → ok.
3. Открыть `http://localhost:8080/` в браузере, увидеть placeholder и кнопку.
4. Открыть DevTools → Network → проверить, что `/api/health` действительно проксируется бэкендом.
5. `ls ./data/` → `app.db`, `app.db-shm`, `app.db-wal`.
6. `docker compose down && docker compose up` → данные сохранились.
7. Dev-режим:
   - В контейнере: остановить compose.
   - `cd backend && go run ./cmd/server` (env: `SESSION_SECRET=test`, `DATA_DIR=./data`).
   - `cd frontend && npm install && npm run dev`.
   - Открыть `http://localhost:5173/` → видим страницу, `/api/health` через прокси работает.
