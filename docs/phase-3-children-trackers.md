# Phase 3 — Children & Trackers

## Цель

После фазы юзер может добавить ребёнка, переключаться между детьми. Админ может создать кастомный трекер (например «Feeding» с полями When+Amount). В UI на странице трекера видна placeholder-форма с уже отрендеренными полями по schema, но без сохранения записей (это Phase 4).

## Объём (Scope)

- Таблицы `children`, `trackers`. Миграция `0003_children_trackers.sql`.
- API CRUD для children (любой залогиненный юзер).
- API CRUD для trackers (только admin).
- Серверная валидация `schema_json`:
  - Корневой объект с массивом `fields`.
  - Каждое поле: `key` (snake_case, уникальный), `label.en` обязателен, `label.uk` опционален, `type` из allowlist, `required` bool, плюс type-specific опции (number: `min/max/unit`; select/multiselect: непустой `options[].value` + `options[].label.en`).
  - Не более одного поля с `isPrimaryTime: true`, оно должно быть `datetime|date|time`.
- UI:
  - `ChildSwitcher.vue` в шапке — выбор активного ребёнка, сохраняется в `localStorage`.
  - Pinia store `children`.
  - `ChildrenAdmin.vue` (страница `/children`): список, create/edit/delete (любой юзер).
  - `TrackerList.vue` (страница `/trackers`): карточки трекеров, кнопка `+ New tracker` (admin only).
  - `TrackerForm.vue` (страница `/trackers/new`, `/trackers/:id/edit`, admin only): name, icon picker (lucide names), color, description, и `SchemaEditor.vue`.
  - `SchemaEditor.vue`: drag-to-reorder список полей, кнопка `+ Add field`, для каждого поля inline-редактирование (key, label EN/UK, type, флаги, type-specific options). Сохранение через emit.
  - `TrackerView.vue` (страница `/trackers/:id`): шапка с именем и иконкой; рендер пустой формы через `DynamicEntryForm` (заглушка submit). Без списка записей.

## Не входит в фазу

- CRUD записей (Phase 4).
- Динамический рендер формы для **сохранения** — только preview (Phase 4 подключит submit).
- Графики/timeline (Phase 8).
- Локализация labels полей в UI — пока показываем `label.en` (Phase 8).

## Зависимости

- Phase 1, 2.

## Изменения в БД

`0003_children_trackers.sql`:

```sql
-- +goose Up
CREATE TABLE children (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  birthday DATE,
  avatar_url TEXT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE trackers (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  icon TEXT,                              -- имя из lucide-vue-next
  color TEXT,                             -- hex
  description TEXT,
  schema_json TEXT NOT NULL,
  is_archived INTEGER NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_trackers_archived ON trackers(is_archived);

-- +goose Down
DROP TABLE trackers;
DROP TABLE children;
```

## Backend — что добавляется

```
backend/internal/
├── children/
│   ├── store.go               # CRUD
│   └── handlers.go
├── trackers/
│   ├── schema.go              # типы FieldDef, Schema; Validate(schemaJSON []byte) error
│   ├── store.go               # CRUD
│   └── handlers.go
└── http/router.go             # /api/children, /api/trackers
```

`schema.go` ключевые типы:

```go
type Schema struct {
    Fields []FieldDef `json:"fields"`
}

type FieldDef struct {
    Key            string                 `json:"key"`
    Label          map[string]string      `json:"label"` // {"en":"...", "uk":"..."}
    Type           string                 `json:"type"`
    Required       bool                   `json:"required"`
    IsPrimaryTime  bool                   `json:"isPrimaryTime,omitempty"`
    // type-specific:
    Unit    string         `json:"unit,omitempty"`
    Min     *float64       `json:"min,omitempty"`
    Max     *float64       `json:"max,omitempty"`
    Options []FieldOption  `json:"options,omitempty"`
    Multiline bool         `json:"multiline,omitempty"` // для text->longtext флаг
}

type FieldOption struct {
    Value string            `json:"value"`
    Label map[string]string `json:"label"`
}

func ValidateSchema(s *Schema) error { /* allowlist типов, уникальные keys, ровно <=1 isPrimaryTime, ... */ }
```

При PATCH трекера, если `schema_json` изменился и в `entries` уже есть записи → возвращаем 200 с предупреждением `{warnings: ["..."]}`, но позволяем сохранить (миграция данных не делается). Это упрощённая модель: пользователь сам отвечает за совместимость.

## Frontend — что добавляется

```
frontend/src/
├── api/
│   ├── children.ts
│   └── trackers.ts
├── stores/
│   ├── children.ts            # state: list, activeId; actions: fetchAll, setActive, create, update, remove
│   └── trackers.ts            # state: list; actions: fetchAll, get, create, update, archive, remove
├── lib/
│   └── schema.ts              # типы FieldDef/FieldOption (зеркало backend), helpers (defaultFieldByType)
├── components/
│   ├── ChildSwitcher.vue
│   ├── trackers/
│   │   ├── SchemaEditor.vue
│   │   ├── FieldEditor.vue     # один ряд поля: key, label EN/UK, type, опции
│   │   ├── FieldRenderer.vue   # рендерит input по type (используется и в форме entry, и в preview)
│   │   └── DynamicEntryForm.vue # обёртка над набором FieldRenderer; в этой фазе — preview без submit
│   └── ui/                     # добавятся: select, switch, textarea, popover, command (для icon picker)
├── views/
│   ├── ChildrenAdmin.vue
│   ├── TrackerList.vue
│   ├── TrackerForm.vue
│   └── TrackerView.vue
└── router/index.ts             # routes
```

`SchemaEditor.vue` UX:
- Список карточек полей, перетаскивание (можно простой up/down кнопками, без drag&drop библиотек на этом этапе).
- Кнопка `+ Add field` → выпадает выбор типа → создаётся новый item с дефолтами.
- Inline-валидация: уникальность `key`, обязательность `label.en`, type-specific.
- На сохранении трекера — schema собирается в JSON и отправляется PATCH/POST.

`ChildSwitcher.vue`:
- Если `children.list.length === 0` → ссылка «Add a child».
- Если 1 ребёнок — показывает имя без выпадушки.
- Если 2+ — dropdown.
- Активный `child_id` хранится в Pinia + sync в `localStorage`.

## API эндпоинты

| Метод | Путь | Доступ | Тело |
|---|---|---|---|
| GET | `/api/children` | user | список |
| POST | `/api/children` | user | `{name, birthday?, avatar_url?}` |
| PATCH | `/api/children/:id` | user | partial |
| DELETE | `/api/children/:id` | admin | каскад `entries.child_id → NULL` |
| GET | `/api/trackers?include_archived=` | user | список |
| GET | `/api/trackers/:id` | user | один |
| POST | `/api/trackers` | admin | `{name, icon?, color?, description?, schema_json}` |
| PATCH | `/api/trackers/:id` | admin | partial; `{warnings:[...]}` если есть entries |
| POST | `/api/trackers/:id/archive` | admin | toggle is_archived |
| DELETE | `/api/trackers/:id` | admin | каскад entries (с подтверждением на фронте) |

## Зависимости (libs)

**Go**: ничего нового — schema валидируется собственным кодом (полная JSON-Schema библиотека подключится в Phase 4 для валидации `data_json`).

**JS**: shadcn-vue components: `select`, `switch`, `textarea`, `popover`, `command`. Установка через `npx shadcn-vue add`.

## Acceptance criteria

- [ ] Юзер создаёт ребёнка, видит его в шапке.
- [ ] Несколько детей — переключатель работает, выбор сохраняется при перезагрузке.
- [ ] Только admin видит кнопку `+ New tracker`.
- [ ] Создаётся трекер «Feeding» с двумя полями: When (datetime, isPrimaryTime, required) и Amount (number, unit=ml, min=0, required).
- [ ] Создаётся трекер «Diaper» с полем Stool color (select со значениями yellow/green/brown).
- [ ] Серверная валидация отвергает schema с дублирующимися `key`, двумя `isPrimaryTime`, неизвестным `type`.
- [ ] PATCH трекера со схемой при наличии entries возвращает warning, но обновление проходит.
- [ ] DELETE трекера на фронте требует подтверждение «удалить трекер и N записей?».
- [ ] DELETE последнего ребёнка не падает: entries остаются, `child_id` становится NULL.
- [ ] Архивный трекер не показывается в `TrackerList` без галочки «Show archived».

## Verification

1. Залогиниться админом, открыть `/children`, добавить «Малыш» с днём рождения.
2. Открыть `/trackers`, нажать `+ New tracker`, создать «Feeding» с полями When+Amount(ml).
3. На странице `/trackers/:id` увидеть форму с двумя полями (форма не сохраняет — это Phase 4).
4. Отредактировать трекер: добавить третье поле Note (longtext, optional). Сохранить.
5. Создать трекер «Diaper» с select-полем «Color» (3 опции).
6. Попробовать создать поле с тем же `key`, что уже есть → форма не даёт сохранить.
7. POST `/api/trackers` через curl с двумя `isPrimaryTime: true` → 400.
8. Создать второго юзера (admin), создать второго ребёнка под обычным юзером.
9. Переключатель в шапке показывает оба, выбор сохраняется в localStorage.
10. Архивировать «Feeding», убедиться что он скрывается.
