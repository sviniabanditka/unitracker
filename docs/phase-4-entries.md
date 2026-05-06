# Phase 4 — Entries & Dynamic Forms

## Цель

После фазы пользователь может полноценно вести трекинг: открыть трекер, заполнить динамическую форму по schema (с типизированными инпутами), сохранить запись, редактировать, удалять. Список записей трекера фильтруется по диапазону дат и активному ребёнку.

## Объём (Scope)

- Таблица `entries`, миграция `0004_entries.sql`.
- Серверная валидация `data_json` по `schema_json` трекера через JSON-Schema (адаптер из Schema → JSON-Schema draft 2020-12).
- Извлечение `occurred_at` из поля с `isPrimaryTime`, fallback на `created_at`.
- API CRUD `/api/trackers/:id/entries` и `/api/entries/:id`.
- Soft delete (`is_deleted=1`); удалённые не возвращаются в list, но доступны по id (для подготовки к Phase 5 revisions).
- UI:
  - `DynamicEntryForm.vue` дописывается до полнофункциональной (vee-validate + zod-схема, сгенерированная по `schema.fields`).
  - `FieldRenderer.vue` поддерживает все типы полей: `datetime/date/time` (нативный `<input>` или shadcn-vue date picker), `number` (с unit-suffix и step), `text/longtext` (input/textarea), `select/multiselect` (shadcn-vue Select / Combobox), `boolean` (Switch), `color` (color input), `duration` (mm:ss или hh:mm).
  - `TrackerView.vue` показывает: «Add entry» CTA → форма (модал или drawer на мобиле), список последних N записей с пагинацией.
  - Клик на запись → drawer/edit-форма.
  - Фильтр по диапазону дат (date range picker) и активный ребёнок применяется автоматически.
  - Pinia store `entries` с per-tracker кэшем.

## Не входит в фазу

- Revisions (Phase 5).
- Dashboard со сводной лентой (Phase 8).
- Графики/агрегации (Phase 8).
- Валидация консистентности при изменении schema трекера задним числом — пишем «как есть», читаем по новой schema (несовместимые поля просто не отрендерятся).

## Зависимости

- Phase 1-3.

## Изменения в БД

`0004_entries.sql`:

```sql
-- +goose Up
CREATE TABLE entries (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tracker_id INTEGER NOT NULL REFERENCES trackers(id) ON DELETE CASCADE,
  child_id INTEGER REFERENCES children(id) ON DELETE SET NULL,
  data_json TEXT NOT NULL,
  occurred_at DATETIME NOT NULL,
  created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
  is_deleted INTEGER NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_entries_tracker_occurred ON entries(tracker_id, occurred_at DESC);
CREATE INDEX idx_entries_child_occurred ON entries(child_id, occurred_at DESC);

-- +goose Down
DROP TABLE entries;
```

## Backend — что добавляется

```
backend/internal/
├── entries/
│   ├── store.go               # CRUD c фильтрами
│   ├── validator.go           # SchemaToJSONSchema(s *trackers.Schema) []byte; ValidateData(jsonSchema, data []byte) error
│   ├── service.go             # Create/Update/SoftDelete с извлечением occurred_at
│   └── handlers.go
└── http/router.go             # /api/trackers/:id/entries, /api/entries/:id
```

`validator.go` — генератор JSON-Schema:
- `datetime/date/time` → `string` + `format`.
- `number` → `number` + `minimum/maximum` если заданы.
- `text` → `string` + `maxLength: 500` дефолт; `longtext` → `maxLength: 10000`.
- `select` → `enum`.
- `multiselect` → `array` с `items.enum` + `uniqueItems`.
- `boolean` → `boolean`.
- `color` → `string` + `pattern: ^#[0-9a-fA-F]{6}$`.
- `duration` → `integer` (миллисекунды) + `minimum: 0`.
- `required` → попадает в `required: []`.
- `additionalProperties: false`.

Скомпилированный `*jsonschema.Schema` кэшируется в map по `tracker.id` + хэш schema_json (invalidate при PATCH трекера).

`service.go` поток create:
1. Получить трекер, скомпилировать его JSON-Schema (или из кэша).
2. Валидировать `data_json` → 400 при ошибке с детализацией.
3. Найти `primaryTimeKey` из schema; если есть и значение валидно → распарсить в `occurred_at`. Иначе `occurred_at = NOW()`.
4. INSERT в транзакции.

## Frontend — что добавляется

```
frontend/src/
├── api/
│   └── entries.ts
├── stores/
│   └── entries.ts             # per-tracker список, фильтры, pagination cursor
├── components/
│   └── trackers/
│       ├── DynamicEntryForm.vue   # production-версия
│       ├── FieldRenderer.vue      # все типы
│       └── EntryListItem.vue      # компактное отображение записи в списке
├── views/
│   └── TrackerView.vue        # дописать: список + Add CTA + filter bar
└── lib/
    ├── zod-from-schema.ts     # zodSchemaFromTrackerSchema(schema) → ZodObject
    └── format.ts              # форматтеры для значений в списке
```

`DynamicEntryForm.vue`:
- props: `schema`, `initial?` (для редактирования), `tracker` (для подсказок).
- Использует vee-validate с `toTypedSchema(zodSchemaFromTrackerSchema(schema))`.
- При `isPrimaryTime` поле — auto-fill текущим временем при создании.
- Прогревочные дефолты по типу: `boolean=false`, `number=null`, `select=options[0].value` если required, `datetime=now()`.
- emit `submit(data)`; обёртка вызывает api.

`EntryListItem.vue`:
- Заголовок: значение primaryTime в формате «HH:mm» если сегодня, «Mon DD, HH:mm» иначе.
- Подзаголовок: 1-2 ключевых поля (число + unit, выбранный select).
- При клике открывается edit-drawer.

## API эндпоинты

| Метод | Путь | Доступ | Параметры |
|---|---|---|---|
| GET | `/api/trackers/:id/entries` | user | `?child_id=&from=&to=&limit=&cursor=` |
| GET | `/api/entries/:id` | user | один (включая удалённые при `?include_deleted=true`) |
| POST | `/api/trackers/:id/entries` | user | `{data: {...}, child_id?: number}` |
| PATCH | `/api/entries/:id` | user | `{data?: {...}, child_id?: number}` |
| DELETE | `/api/entries/:id` | user | soft delete |

`from`/`to` — ISO8601, `cursor` — base64 от `(occurred_at, id)`. По умолчанию `limit=50`.

Ответ list:
```json
{
  "items": [{"id":1, "tracker_id":1, "child_id":1, "data":{...}, "occurred_at":"...", "created_at":"...", "updated_at":"..."}],
  "next_cursor": "..."
}
```

## Зависимости (libs)

**Go**: `github.com/santhosh-tekuri/jsonschema/v6`.

**JS**: `vee-validate`, `@vee-validate/zod`, `zod`, `date-fns`. shadcn-vue: `dialog`, `drawer`, `calendar`, `range-calendar` (или date picker), `combobox`.

## Acceptance criteria

- [ ] В трекере «Feeding» создаются записи через форму. occurred_at = значение поля When.
- [ ] Запись с `amount=-1` отвергается сервером (min=0) и подсвечивается на фронте.
- [ ] Запись без обязательного поля невозможна (zod не даёт submit).
- [ ] Список записей пагинируется по 50, cursor работает.
- [ ] Фильтр по диапазону дат сужает список.
- [ ] Переключение ребёнка в шапке мгновенно перефильтровывает список (если `child_id` задан в URL/store).
- [ ] Edit-drawer редактирует existing entry, после сохранения список обновляется.
- [ ] Soft delete: запись пропадает из списка, в БД `is_deleted=1`.
- [ ] Удаление трекера (Phase 3) каскадно удаляет entries (включая soft-deleted).
- [ ] Изменение schema трекера не ломает старые записи: если в старой записи нет нового required-поля, она просто отображается без него (предупреждение в edit-форме).

## Verification

1. Открыть трекер «Feeding», добавить 5 записей с разным временем и количеством.
2. Открыть трекер «Diaper», добавить запись с цветом yellow.
3. Создать второго ребёнка, переключиться, добавить ещё запись Feeding (идёт под второго ребёнка).
4. Переключаясь в шапке между детьми — список меняется.
5. Применить фильтр «Last 7 days» → подходящие записи остаются.
6. Отредактировать одну запись (изменить amount), сохранить — пересортировка не нужна, но обновление видно.
7. Удалить запись → исчезает из списка.
8. `curl POST /api/trackers/1/entries -d '{"data":{"amount":-5}}'` → 400 с деталями.
9. `curl POST /api/trackers/1/entries -d '{"data":{}}'` → 400, перечислены missing required fields.
10. SQL-проверка: `SELECT id, occurred_at, json_extract(data_json,'$.amount') FROM entries;` — данные на месте.
