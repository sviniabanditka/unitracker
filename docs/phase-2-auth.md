# Phase 2 — Auth & Users

## Цель

После фазы можно зайти в систему под bootstrap-админом из env, выйти, сменить пароль, создать обычного пользователя, войти под ним. Всё закрытое API защищено middleware. Публичных эндпоинтов кроме `/api/auth/login` и `/api/health` нет.

## Объём (Scope)

- Таблицы `users` и `sessions`, миграция `0002_auth.sql`.
- Bootstrap первого админа: при старте, если `users` пустая и заданы `INITIAL_ADMIN_USERNAME`/`INITIAL_ADMIN_PASSWORD`, создаётся админ. При повторном старте ничего не пересоздаётся.
- Эндпоинты `POST /api/auth/login`, `POST /api/auth/logout`, `GET /api/auth/me`.
- Серверные сессии: id — random 32 bytes (base64url), хранится в `sessions(id)`, secure httpOnly cookie `session_id`. TTL — 30 дней, `last_used_at` обновляется при каждом запросе (опц.) или хотя бы раз в сутки.
- Middleware:
  - `RequireUser` — кладёт `User` в контекст, иначе 401.
  - `RequireAdmin` — то же + проверка роли, иначе 403.
- Эндпоинты админа: `GET/POST/PATCH/DELETE /api/admin/users` (CRUD).
- Эндпоинт `POST /api/auth/change-password` для текущего юзера (старый пароль + новый).
- UI:
  - `Login.vue`, форма username/password.
  - Pinia store `auth` с `me`, `login`, `logout`, `changePassword`.
  - Router-guard в `router/index.ts`: если не залогинен → редирект на `/login`; если залогинен и зашёл на `/login` → редирект на `/`.
  - Layout с шапкой: имя пользователя, роль, кнопка logout. Если admin — линк на `/admin`.
  - Страница `admin/UsersAdmin.vue`: таблица юзеров, кнопки create/edit/delete/reset-password.
  - Страница `Account.vue`: смена своего пароля.

## Не входит в фазу

- Дети, трекеры, записи (Phase 3-4).
- 2FA, OAuth, password recovery email — не нужны для self-hosted семейной установки.
- Rate limit на login — добавим если будет угроза публичной экспозиции (Phase 8 polish, опц.).
- Audit-лог входов.

## Зависимости

- Phase 1 (Foundation).

## Изменения в БД

`backend/migrations/0002_auth.sql`:

```sql
-- +goose Up
CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username TEXT NOT NULL UNIQUE COLLATE NOCASE,
  password_hash TEXT NOT NULL,
  role TEXT NOT NULL CHECK (role IN ('admin','user')),
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sessions (
  id TEXT PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at DATETIME NOT NULL,
  last_used_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- +goose Down
DROP TABLE sessions;
DROP TABLE users;
```

## Backend — что добавляется

```
backend/internal/
├── auth/
│   ├── password.go            # bcrypt Hash, Compare; cost=12
│   ├── session.go             # Store: Create, Get, Delete, DeleteByUser, GC
│   ├── middleware.go          # RequireUser, RequireAdmin; ctx ключ User
│   ├── handlers.go            # Login, Logout, Me, ChangePassword
│   └── bootstrap.go           # BootstrapAdminFromEnv(db)
├── users/
│   ├── store.go               # CRUD по users
│   └── handlers.go            # admin endpoints
└── http/router.go             # подключает /api/auth/*, /api/admin/users/*
```

`session.go`:
- id — `crypto/rand` 32 байта → `base64.URLEncoding.WithoutPadding`.
- При логине удаляются истёкшие сессии текущего пользователя.
- `GC` запускается раз в час: `DELETE FROM sessions WHERE expires_at < ?`.

`middleware.go`:
- Читает cookie `session_id`, ищет в БД, проверяет `expires_at > now`.
- Кладёт User в context; ответ 401 при отсутствии/истёкшей.
- `RequireAdmin` после `RequireUser` проверяет `role == 'admin'`.

`bootstrap.go` запускается из `main.go` после миграций:
```
if count(users) == 0 && env INITIAL_ADMIN_USERNAME && env INITIAL_ADMIN_PASSWORD {
  create admin
} else if count(users) == 0 {
  log fatal "no admin and no INITIAL_ADMIN_* env"
}
```

## Frontend — что добавляется

```
frontend/src/
├── api/
│   ├── auth.ts                # login, logout, me, changePassword
│   └── users.ts               # admin endpoints
├── stores/
│   └── auth.ts                # state: me|null; actions: ensure(), login, logout, changePassword
├── views/
│   ├── Login.vue
│   ├── Account.vue            # сменить пароль
│   └── admin/
│       └── UsersAdmin.vue
├── components/
│   ├── AppLayout.vue          # header (user, role badge, logout) + slot
│   └── ui/                    # shadcn-vue: button, input, label, dialog, table, badge — добавить через npx shadcn-vue add
└── router/index.ts            # routes: /login, /, /account, /admin/users; guards
```

Pinia store `auth.ts`:
- `state: { me: User | null, ready: boolean }`.
- `ensure()` — если `!ready`, делает `GET /api/auth/me`, при 401 ставит `me=null`, `ready=true`.
- `login`/`logout` обновляют `me`.
- В `App.vue` на mount → `auth.ensure()`.
- Router `beforeEach`: ждёт `ready`; если route требует auth и `me==null` → `/login`; если route требует admin и `me.role!='admin'` → `/`.

## API эндпоинты

| Метод | Путь | Доступ | Тело / параметры |
|---|---|---|---|
| POST | `/api/auth/login` | public | `{username, password}` → `{user}` + cookie |
| POST | `/api/auth/logout` | user | удаляет сессию + clear cookie |
| GET | `/api/auth/me` | user | `{user}` или 401 |
| POST | `/api/auth/change-password` | user | `{old_password, new_password}` |
| GET | `/api/admin/users` | admin | список |
| POST | `/api/admin/users` | admin | `{username, password, role}` |
| PATCH | `/api/admin/users/:id` | admin | `{username?, role?, new_password?}` |
| DELETE | `/api/admin/users/:id` | admin | self-delete запрещён, последнего админа удалить запрещено |

## Зависимости (libs)

**Go**: `golang.org/x/crypto/bcrypt`.

**JS**: ничего нового кроме shadcn-vue компонентов, добавляемых через CLI.

## Acceptance criteria

- [ ] При первом запуске с заданными `INITIAL_ADMIN_USERNAME`/`PASSWORD` создаётся админ.
- [ ] При повторном запуске admin не пересоздаётся, env-пара игнорируется (warn в логах если задана).
- [ ] При отсутствии `INITIAL_ADMIN_*` и пустой `users` приложение падает с понятной ошибкой.
- [ ] Login возвращает cookie с флагами `HttpOnly`, `SameSite=Lax`, `Secure` (в prod) и `Path=/`.
- [ ] Открытие любой страницы кроме `/login` без сессии редиректит на `/login`.
- [ ] После logout сессия удалена из БД, cookie очищена.
- [ ] Юзер не может зайти в `/admin/*` (фронт + бэк).
- [ ] Админ может создать обычного юзера, обычный юзер логинится.
- [ ] Смена пароля своего работает, после неё все остальные сессии этого юзера удаляются.
- [ ] Удаление последнего админа возвращает 400.
- [ ] GC удаляет истёкшие сессии (проверить уменьшив TTL).

## Verification

1. `INITIAL_ADMIN_USERNAME=admin INITIAL_ADMIN_PASSWORD=admin123 docker compose up --build`.
2. Открыть `/` → редирект на `/login`.
3. Войти как `admin/admin123` → попадаем на `/`. Шапка показывает `admin` + badge `admin`.
4. Перейти `/admin/users` → таблица из одного юзера.
5. Создать юзера `alice/alice123` с ролью `user`.
6. Logout → редирект на `/login`. Зайти как `alice/alice123`.
7. Проверить: `/admin/users` отвечает 403/редирект на `/`.
8. На странице `/account` сменить пароль `alice123 → alice456`. Logout/login с новым.
9. `curl -X GET http://localhost:8080/api/auth/me` без cookie → 401.
10. `docker compose down && docker compose up` → юзеры на месте, env `INITIAL_*` игнорируется (в логе warning).
11. Удалить `data/app.db`, перезапустить без env `INITIAL_*` → приложение падает с явной ошибкой про отсутствие админа.
