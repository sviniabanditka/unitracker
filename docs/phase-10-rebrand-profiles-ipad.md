# Phase 10 — Profiles, scoped trackers, per-user access, iPad-tier responsive

## Цель

Привести модель к мульти-профильной (`children` → `profiles`), сделать трекеры
индивидуальными для профиля, добавить admin-управляемый per-user-доступ к
профилям и аккуратный третий брейкпоинт под iPad между mobile и desktop.

## Scope

1. **Rebrand entity**
   - `children` → `profiles`. Поля: `id, name, avatar_url, description, created_at, updated_at`.
   - PWA / `<title>` / `app.name` → **Tracker**. localStorage-ключ `baby-tracker:active-child` мигрирует one-shot в `tracker:active-profile`.

2. **Profile-scoped trackers**
   - `trackers.profile_id` (FK на `profiles`, ON DELETE CASCADE; nullable на схеме, NOT NULL по контракту в Go).
   - `/api/profiles/{pid}/trackers` (GET/POST), `/api/trackers/library` (GET все доступные с `profile_name`).
   - `entries.profile_id` всегда выводится из tracker'а на стороне сервера; клиент `profile_id` в body не передаёт.
   - В TrackerForm — кнопка **Clone from library** → диалог → prefill `name/icon/color/description/schema_json` (deep clone, новые entries не переносятся).

3. **Per-user profile access**
   - Таблица `user_profile_access (user_id, profile_id, granted_at, granted_by)`.
   - Admin: неявный доступ ко всем (по `users.role='admin'`); записей в таблице у admin'ов нет.
   - Member: доступ только к профилям из таблицы.
   - Backfill: существующие member'ы получают полный доступ ко всем существующим профилям (preserves поведение до Phase 10).
   - Admin endpoints: `GET / PUT /api/admin/users/{uid}/profiles`.
   - Middlewares (`access.RequireProfileAccess`, `access.RequireProfileAccessFromQuery`, `access.RequireTrackerAccess`, локальный `requireEntryAccess` в router) возвращают 404 на запрещённый ресурс — не раскрывают существование.

4. **iPad-tier responsive (mobile < md / tablet md..<lg / desktop lg+)**
   - `AppLayout` контейнер `max-w-4xl lg:max-w-6xl`. Бургер-drawer остаётся под md.
   - `RowActions` — gear-меню до `lg:`, inline-кнопки от `lg:`. Раньше split был на `md:`.
   - `TrackerList` `grid sm:grid-cols-2 lg:grid-cols-3`.
   - `Dashboard` `max-w-3xl lg:max-w-5xl` + `lg:grid lg:grid-cols-2` на секциях today / last_per_tracker.
   - `ChartsSection` cards `grid-cols-1 md:grid-cols-2 lg:grid-cols-3`. CalendarHeatmap `lg:col-span-3`.
   - Admin views: `max-w-3xl lg:max-w-5xl`.
   - `TrackerIconPicker` `grid-cols-8 sm:grid-cols-10 lg:grid-cols-12`.

## Не входит

- Перенос Go module path или переименование репо-каталога.
- Sidebar-навигация на tablet+. Горизонтальный nav остаётся.
- Granular per-tracker access — доступ выдаётся только на профиль целиком.
- Self-service запрос доступа.
- Move-tracker-between-profiles. `profile_id` immutable у трекера.
- Backfill старых trackers/entries в новую модель — wipe вместо backfill (по запросу).

## БД

`backend/migrations/0008_profiles.sql` (Up):
1. `ALTER TABLE children RENAME TO profiles`; drop `birthday`; add `description`.
2. `DELETE FROM trackers` — каскадно зачищает entries/entry_revisions.
3. Rename `entries.child_id → profile_id`, индекс `idx_entries_profile_occurred`.
4. Rename `entry_revisions.child_id → profile_id` (без FK, snapshot-колонка).
5. Add `trackers.profile_id` (FK на `profiles`, CASCADE; nullable на схеме). Index `idx_trackers_profile`.
6. Create `user_profile_access` + indexes.
7. Backfill: каждому user'у с role='user' выдать доступ ко всем существующим profiles.
8. `UPDATE settings` менять `app_name` "Baby Tracker" → "Tracker".

## Backend

- Новый пакет `internal/access/`: `Store.{ListProfileIDs, ListGrantedProfileIDs, HasAccess, Replace}`, handlers `MountAdminUserAccess`, middlewares `RequireProfileAccess`, `RequireTrackerAccess`, `RequireProfileAccessFromQuery`.
- Перенос `internal/children/` → `internal/profiles/`. Тип `Profile`, `ProfilesStore`. List фильтруется через `AccessLister` (admin → все, member → grants).
- `trackers/store.go`: `ListByProfile`, `Library` (JOIN profiles for profile_name), `GetProfileID`. `Tracker` struct +`ProfileID`. `Create(input)` принимает `ProfileID`.
- `trackers/handlers.go`: `MountProfileScoped` + `MountProfileScopedAdmin`, `MountLibrary`, `MountTrackerScoped` + `MountTrackerScopedAdmin`. PATCH игнорирует `profile_id`.
- `entries/`: `ChildID` → `ProfileID` повсеместно. На create entry: `profile_id` берётся из tracker, body ignored. ListFilter без `ChildID`. `GetProfileIDForEntry` (для middleware).
- `dashboard_handler.go`: `profile_id` обязателен; trackers фильтруются `ListByProfile(pid, false)`.
- `stats.go`: `child_id` query param выпилен (tracker pins profile).
- `http/router.go`: переписан под профильно-scoped routes, mount `/api/admin/users/{uid}/profiles`, гейт по middlewares. Старые `/api/children`, `/api/trackers` (list/create) удалены.

## Frontend

- `api/profiles.ts`, `stores/profiles.ts` (с one-shot localStorage migration), `components/ProfileSwitcher.vue`, `views/ProfilesAdmin.vue`. Удалены старые `children.ts`, `ChildSwitcher.vue`, `ChildrenAdmin.vue`.
- `api/trackers.ts`: `list({profileId})`, `library()`, `create(profileId, ...)`. Тип `Tracker.profile_id`.
- `stores/trackers.ts`: `byProfile`, `library`, `loadLibrary()`, `ensure(profileId)`.
- `api/entries.ts`, `dashboard.ts`, `stats.ts`, `revisions.ts` — `child_id`/`childId` → `profile_id`/`profileId` в типах. Из payload запросов (create/update entry) убрано.
- `components/trackers/TrackerLibraryDialog.vue` — модалка с поиском и списком (TrackerIcon + name + profile_name).
- `views/TrackerForm.vue` — кнопка "Clone from library" в create-mode → prefill полей. Save POST'ит в `/profiles/{activeProfileId}/trackers`.
- `views/TrackerView.vue` — watcher на `activeId`: если tracker.profile_id ≠ activeId, redirect `/trackers`.
- `views/TrackerList.vue` — рендерит `byProfile[activeId]`, empty state "Создайте профиль" / "Свяжитесь с админом".
- `api/admin/userAccess.ts` + `components/users/UserProfilesDialog.vue` — admin-флоу управления per-user grants.
- `views/admin/UsersAdmin.vue` — RowAction "Manage profiles" (только для member'ов).
- `i18n/{en,uk}.json` — блок `profiles.*`, новые ключи (`trackers.cloneFromLibrary`, `users.manageProfiles`, `profiles.noAccess` …), `app.name = "Tracker"`.
- `vite.config.ts`, `index.html`, `package.json` — branding.

## API смешанный обзор

- `GET /api/profiles` — отфильтрованный по доступу.
- `POST /api/profiles` — admin only.
- `PATCH /api/profiles/{id}` — admin only (фактически в текущей UX даже member не запускает; Mount гейтит по RequireAdmin).
- `DELETE /api/profiles/{id}` — admin only.
- `GET /api/profiles/{pid}/trackers` — gated by `RequireProfileAccess`.
- `POST /api/profiles/{pid}/trackers` — admin + access.
- `GET /api/trackers/library` — фильтрован по доступу пользователя.
- `GET /api/trackers/{tid}` — gated by `RequireTrackerAccess`.
- `PATCH /api/trackers/{tid}` — admin + access.
- `POST /api/trackers/{tid}/archive`, `DELETE /api/trackers/{tid}` — admin + access.
- `GET/POST /api/trackers/{tid}/entries`, `GET /api/trackers/{tid}/stats` — access-gated.
- `GET/PATCH/DELETE /api/entries/{id}`, `GET /api/entries/{id}/revisions`, `POST .../restore/{rid}`, `POST .../restore-deleted` — gated `requireEntryAccess`.
- `GET /api/dashboard?profile_id=X` — `RequireProfileAccessFromQuery`.
- `GET / PUT /api/admin/users/{uid}/profiles` — admin only.

## Verification (короткая выжимка)

- `cd backend && go build ./... && go vet ./...` — clean.
- `sqlite3 .data/app.db ".schema profiles trackers entries user_profile_access"` — структура соответствует.
- Curl-смок (admin + member, см. план) подтверждает: profile create/list, trackers per profile, library с profile_name, derive profile_id на create entry, 404 для члена без grant'а, 200 после grant'а, 403 на admin endpoints для member'а.
- `cd frontend && vue-tsc -p tsconfig.app.json --noEmit` — 0 errors. `npm run build` — clean.
- DevTools manifest → "Tracker", `<title>` "Tracker".
- Manual: localStorage migration, clone-from-library flow, admin grant→member видит профиль, revoke→member теряет доступ.
- Responsive matrix: 390 / 820 / 1180 / 1440 px, без горизонтального скролла, бургер↔inline на md, RowActions gear↔inline на lg, TrackerList 1/2/3 кол.
