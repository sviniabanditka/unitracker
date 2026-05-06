# Phase 8 — i18n, PWA, Mobile Polish

## Цель

Финальная фаза: интерфейс полностью локализован (EN + UK с переключателем), приложение устанавливается на домашний экран мобильного телефона как PWA и работает с offline-shell, навигация удобна с одной руки. Появляется Dashboard со сводной лентой последних записей всех трекеров.

## Объём (Scope)

### i18n
- `vue-i18n` подключён, файлы `src/i18n/{en,uk}.json`.
- Все статические строки UI (кнопки, заголовки, ошибки форм, badges) вынесены в i18n.
- LocaleSwitcher в шапке: EN | UK; выбор сохраняется в `localStorage.locale`. Если нет — берётся `default_locale` из settings.
- Локализация labels трекеров и опций select: на странице используется `label[locale] || label.en`.
- Ошибки бэкенда возвращают стабильный `code` (например `"validation.required"`) — фронт мапит в i18n.
- Форматирование дат через `date-fns` + локали `enUS` / `uk`.
- Pluralization для счётчиков («3 entries» / «3 записи»).

### PWA
- `vite-plugin-pwa` сконфигурирован (`registerType: 'autoUpdate'`).
- Manifest: `name`, `short_name`, `theme_color`, `background_color`, `display: standalone`, `start_url: /`, иконки 192/512/maskable.
- Service worker: precache shell (CSS/JS), runtime cache для `/api/auth/me` (NetworkFirst, fallback из кэша) и иконок.
- API-данные **не** кэшируются для записи; offline-режим только для просмотра последних загруженных страниц.
- Update prompt: при появлении новой версии SW — toast «Update available» с кнопкой Reload.

### Mobile polish
- Bottom navigation bar на mobile (`md:hidden`): Home / Trackers / + (quick-add) / Account / Admin (если admin).
- Полная адаптивность: формы — drawer на mobile, dialog на desktop.
- Quick-add (`+`): floating button, открывает выбор трекера → форма entry.
- Touch-friendly размеры (минимум 44×44 px для кнопок).
- Тёмная тема через Tailwind `dark:` (системная prefers-color-scheme + toggle в шапке).

### Dashboard
- Новая страница `/` (заменяет placeholder из Phase 1).
- Виджеты:
  - **Today timeline** — лента всех записей за сегодня по всем трекерам, активный ребёнок, отсортировано по времени.
  - **Last entry per tracker** — карточка для каждого трекера с временем последней записи и значением primary поля.
  - **Quick add** — кнопки «+ Feeding», «+ Diaper», … (топ-3 трекера по частоте за 7 дней).

## Не входит в фазу

- Графики/агрегации по неделям (можно как Phase 9 если понадобится).
- Push-нотификации.
- Экспорт CSV.
- Ассистент-сводки.

## Зависимости

- Phase 1-7.

## Изменения в БД

Нет.

## Backend — что добавляется / изменяется

```
backend/internal/
├── http/
│   └── errors.go              # унифицированный error response: {code, message, fields?}; стабильные коды
└── entries/
    └── handlers.go            # новый эндпоинт GET /api/dashboard
```

`GET /api/dashboard?child_id=&date=YYYY-MM-DD` возвращает:
```json
{
  "today": [
    {"tracker_id":1, "tracker_name":"Feeding", "tracker_icon":"baby-bottle", "id":42, "occurred_at":"...", "data":{...}}
  ],
  "last_per_tracker": [
    {"tracker_id":1, "entry": {...} | null}
  ],
  "top_trackers": [1, 3, 2]
}
```

Унификация ошибок (рефактор Phase 1-7):
```json
{
  "code": "validation.failed",
  "message": "Validation failed",
  "fields": {"amount": "must be >= 0"}
}
```
Bash-коды: `auth.unauthorized`, `auth.forbidden`, `validation.failed`, `validation.required`, `not_found`, `conflict`, `maintenance`, `internal`.

## Frontend — что добавляется / изменяется

```
frontend/src/
├── i18n/
│   ├── index.ts               # createI18n, локали, fallback en
│   ├── en.json
│   └── uk.json
├── components/
│   ├── LocaleSwitcher.vue
│   ├── ThemeToggle.vue
│   ├── BottomNav.vue          # mobile only
│   ├── QuickAddFab.vue
│   └── dashboard/
│       ├── TodayTimeline.vue
│       ├── LastEntriesGrid.vue
│       └── QuickAddBar.vue
├── views/
│   └── Dashboard.vue          # / (replaces Home placeholder)
├── lib/
│   ├── locale.ts              # resolveLabel(label, locale)
│   ├── format-date.ts         # i18n-aware форматирование
│   └── error-mapping.ts       # бэкендовые codes → i18n keys
├── vite.config.ts             # plugin-pwa config
└── public/
    ├── icon-192.png
    ├── icon-512.png
    └── icon-maskable-512.png
```

Структура `en.json` (фрагмент):
```json
{
  "app": {"name": "Baby Tracker"},
  "auth": {"login": "Sign in", "logout": "Sign out", "username": "Username", "password": "Password"},
  "common": {"save": "Save", "cancel": "Cancel", "delete": "Delete", "edit": "Edit"},
  "trackers": {"new": "New tracker", "schema": "Schema"},
  "entries": {"add": "Add entry", "show_deleted": "Show deleted"},
  "validation": {
    "required": "This field is required",
    "min": "Minimum value is {min}"
  },
  "errors": {
    "auth.unauthorized": "Please sign in",
    "maintenance": "Server is updating, please wait..."
  }
}
```

`uk.json` — параллельный.

`resolveLabel(label, locale)`:
```ts
export function resolveLabel(label: Record<string,string>|string, locale: string): string {
  if (typeof label === 'string') return label
  return label[locale] || label.en || Object.values(label)[0] || ''
}
```

`vite.config.ts` PWA-секция:
```ts
VitePWA({
  registerType: 'autoUpdate',
  manifest: {
    name: 'Baby Tracker', short_name: 'BabyTracker',
    description: 'Self-hosted baby tracker',
    theme_color: '#0ea5e9', background_color: '#ffffff',
    display: 'standalone', start_url: '/',
    icons: [
      { src: '/icon-192.png', sizes: '192x192', type: 'image/png' },
      { src: '/icon-512.png', sizes: '512x512', type: 'image/png' },
      { src: '/icon-maskable-512.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' }
    ]
  },
  workbox: {
    navigateFallback: '/index.html',
    runtimeCaching: [{
      urlPattern: /\/api\/auth\/me$/,
      handler: 'NetworkFirst',
      options: { cacheName: 'me', networkTimeoutSeconds: 3 }
    }]
  }
})
```

## API эндпоинты

| Метод | Путь | Доступ | Описание |
|---|---|---|---|
| GET | `/api/dashboard` | user | сводка для главной |

Все остальные эндпоинты обновляют формат ошибок до унифицированного.

## Зависимости (libs)

**Go**: ничего нового (унификация ошибок — внутренний рефактор).

**JS**: `vue-i18n@^9`, `vite-plugin-pwa`, `workbox-window`. Иконки сгенерировать (например через https://realfavicongenerator.net) — файлы кладутся вручную в `public/`.

## Acceptance criteria

- [ ] Переключатель локали EN ↔ UK мгновенно меняет интерфейс.
- [ ] Labels полей трекера и опций select показываются на текущей локали (с fallback на `en`).
- [ ] Ошибки валидации с бэка отображаются на текущей локали.
- [ ] Установка PWA на iOS Safari (Add to Home Screen) и Chrome Android (Install app) работает, запуск с иконки открывает в standalone-режиме.
- [ ] При оффлайне можно открыть последнюю посещённую страницу (offline shell), API-запросы фейлятся с offline-баннером.
- [ ] При деплое новой версии в браузере появляется toast «Update available» → клик Reload загружает новую.
- [ ] Bottom-nav виден только на mobile, скрыт на ≥768px.
- [ ] Quick-add FAB добавляет запись за 2 тапа.
- [ ] Dashboard показывает today timeline, последние записи, quick-add бар.
- [ ] Тёмная тема переключается, состояние сохраняется в `localStorage`.
- [ ] Login-страница локализована и подхватывает `default_locale` из settings до выбора пользователя.

## Verification

1. Открыть приложение в desktop, переключить на UK → все строки на украинском, форма создания трекера тоже.
2. Создать трекер с label `{"en":"Amount","uk":"Кількість"}` → на странице entries label соответствует локали.
3. Послать невалидный entry → текст ошибки локализован.
4. На телефоне (Chrome Android) открыть URL → меню «Install app» → установить → запустить с иконки → открывается standalone, есть splash.
5. Включить airplane mode → открыть приложение → loaded shell отображается, баннер «Offline» в шапке.
6. Развернуть новую версию → перезагрузить страницу с открытым приложением → toast «Update available» → Reload.
7. На mobile bottom-nav виден; на ≥md (resize окна) исчезает.
8. Тёмная тема: переключить, проверить что цвета карточек и форм адекватны, обновить страницу → тема сохраняется.
9. Dashboard: добавить пару entries сегодня → они появляются в `Today timeline`.
10. Lighthouse audit (Chrome DevTools) → PWA score ≥ 90, A11y ≥ 90 на главной.

## Завершающие шаги (после Phase 8)

- Обновить `docs/README.md`: пометить все фазы как `[x]`.
- Создать `README.md` в корне проекта с кратким описанием, скриншотами (опц.), инструкциями `docker compose up`.
- Тег `v1.0.0` в git.
