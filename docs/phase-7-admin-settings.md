# Phase 7 — Admin Settings

## Цель

Админ управляет глобальными настройками из UI: интервал авто-снапшотов, политика retention, имя приложения, дефолтная локаль. Изменение интервала **немедленно перепланирует** scheduler без рестарта контейнера. Дефолтная локаль применяется к новым сессиям (если у юзера нет своего выбора).

## Объём (Scope)

- API `/api/admin/settings` (GET / PATCH).
- Типизированные геттеры в `internal/settings`:
  - `BackupIntervalHours() float64`
  - `BackupRetentionCount() int`
  - `AppName() string`
  - `DefaultLocale() string`
- Валидация значений на бэке (interval ≥ 0.05, retention ≥ 1, locale ∈ {en,uk}, app_name length ≤ 64).
- Hot-reload scheduler: в `settings.Store.Set` бросается событие, `backup.Scheduler.Reschedule()` подписан.
- UI:
  - `admin/SettingsAdmin.vue` — простая форма с группировкой:
    - **Backups** — interval (часы, число с шагом 0.5), retention count.
    - **General** — app_name, default_locale (radio en/uk).
  - Сохранение PATCH-ом всех изменённых полей разом.
  - В шапке `app_name` подтягивается из API (через `/api/auth/me` либо отдельный `/api/public/info`).

## Не входит в фазу

- Сложные роли/permissions.
- Time zone setting (используем `TZ` env).
- SMTP/notifications.
- Theme settings (светлая/тёмная — Phase 8 polish).

## Зависимости

- Phase 1-6 (settings таблица создана в Phase 6).

## Изменения в БД

Нет (таблица создана в Phase 6).

## Backend — что добавляется / изменяется

```
backend/internal/
├── settings/
│   ├── store.go           # дописать: типизированные геттеры, Subscribe(callback)
│   └── handlers.go        # GET/PATCH /api/admin/settings
└── backup/
    └── scheduler.go       # подписка: settings.Subscribe("backup_interval_hours", reschedule)
```

`store.Subscribe(key string, fn func(value string))` — простой in-memory pub/sub. При `Set(key, val)` вызывает всех подписчиков `key`.

PATCH принимает `{[key: string]: string}`, валидирует только известные ключи (allowlist), записывает изменения в одной транзакции, после коммита — оповещает подписчиков.

## Frontend — что добавляется

```
frontend/src/
├── api/
│   └── settings.ts        # getAll, patch
├── stores/
│   └── settings.ts        # state: map<key,value>; loaded; actions fetch, save
├── views/admin/
│   └── SettingsAdmin.vue
├── components/admin/
│   └── SettingsForm.vue   # форма с двумя секциями
```

Глобальное использование:
- `app_name` — реактивно в шапке (`AppLayout.vue` подписан на `settings.appName`).
- `default_locale` — применяется в `i18n` если у юзера нет своего выбора (`localStorage.locale`).

## API эндпоинты

| Метод | Путь | Доступ | Тело |
|---|---|---|---|
| GET | `/api/admin/settings` | admin | `{key: value, ...}` все настройки |
| PATCH | `/api/admin/settings` | admin | `{key: value, ...}` partial update; ошибки агрегируются |
| GET | `/api/public/info` | public | `{app_name, default_locale}` для login-страницы |

## Зависимости (libs)

Ничего нового.

## Acceptance criteria

- [ ] Изменение `backup_interval_hours` через UI немедленно перепланирует scheduler (видно в логах `scheduler: rescheduled to 2.0h`).
- [ ] Невалидные значения (interval=0, retention=-1, locale=fr) отвергаются с понятной ошибкой по полю.
- [ ] `app_name` отображается в шапке, обновляется без рефреша после PATCH.
- [ ] `default_locale` применяется на login-странице (видно через `/api/public/info`).
- [ ] PATCH с одним полем не сбрасывает остальные.
- [ ] Юзер не видит `/admin/settings` (как и в Phase 2 admin guard).

## Verification

1. Войти админом, открыть `/admin/settings`.
2. Изменить `app_name` → шапка обновляется.
3. Изменить `default_locale` на `uk` → выйти → на `/login` интерфейс на украинском (после Phase 8; пока — placeholder, но `/api/public/info` возвращает `uk`).
4. Изменить `backup_interval_hours` с 6 на 0.05 → в логах сообщение о rescheduled → через 3 минуты новый авто-снапшот.
5. Поставить `backup_retention_count=2` → создать ещё авто-снапшот → старые удаляются.
6. PATCH с `backup_interval_hours: 0` → 400 с message «must be ≥ 0.05».
7. PATCH с `backup_interval_hours: 12, app_name: "Family Tracker"` одним запросом → оба сохранены.
8. Без admin: `curl /api/admin/settings` → 403.
