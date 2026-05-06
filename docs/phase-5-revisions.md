# Phase 5 — Entry Revisions

## Цель

Каждое изменение записи (создание, редактирование, удаление) фиксируется в `entry_revisions`. На странице записи доступна вкладка «История»: видны все ревизии с автором и временем, можно откатить запись к любой предыдущей версии или восстановить удалённую.

## Объём (Scope)

- Таблица `entry_revisions`, миграция `0005_entry_revisions.sql`.
- Сервис `entries.service.go` дописан: каждый `Create/Update/SoftDelete/Restore` пишет revision в той же транзакции.
- Эндпоинты:
  - `GET /api/entries/:id/revisions` — список ревизий.
  - `POST /api/entries/:id/restore/:revision_id` — заменить текущее состояние на состояние из ревизии (новое revision с `change_type='restore'`).
  - `POST /api/entries/:id/restore-deleted` — снять soft-delete (если запись deleted=1) и записать revision с `change_type='restore'`.
- UI:
  - `EntryHistory.vue` — список ревизий, для каждой: время, автор, тип (badge), мини-diff на ключевых полях.
  - На странице edit-entry — табы `Current` / `History`.
  - Кнопка «Restore this version» в строке ревизии с подтверждением.
  - Список entries (Phase 4) добавляет toggle «Show deleted» (admin-only? нет — любому юзеру), удалённые показываются серым, у каждой — кнопка «Restore».

## Не входит в фазу

- Полный diff-viewer (показываем достаточно информативный summary, но не построчный JSON-diff).
- Permanent delete (удаление revisions) — будет частично решено на уровне retention для snapshots, но revisions храним вечно. Если станет проблемой по объёму — отдельная фаза cleanup.

## Зависимости

- Phase 1-4.

## Изменения в БД

`0005_entry_revisions.sql`:

```sql
-- +goose Up
CREATE TABLE entry_revisions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  entry_id INTEGER NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
  data_json TEXT NOT NULL,
  occurred_at DATETIME NOT NULL,
  child_id INTEGER,
  is_deleted INTEGER NOT NULL DEFAULT 0,
  change_type TEXT NOT NULL CHECK (change_type IN ('create','update','delete','restore')),
  changed_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
  changed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_entry_revisions_entry ON entry_revisions(entry_id, changed_at DESC);

-- +goose Down
DROP TABLE entry_revisions;
```

**Backfill**: при миграции для существующих записей создаётся одна `create`-ревизия с `changed_at = entries.created_at` и `changed_by = entries.created_by`. Делаем это в SQL-миграции:

```sql
INSERT INTO entry_revisions (entry_id, data_json, occurred_at, child_id, is_deleted, change_type, changed_by, changed_at)
SELECT id, data_json, occurred_at, child_id, is_deleted, 'create', created_by, created_at FROM entries;
```

## Backend — что добавляется

```
backend/internal/entries/
├── revisions_store.go         # AppendRevision (в той же tx), List(entry_id), Get(revision_id)
└── service.go                 # обёртки Create/Update/SoftDelete/Restore теперь делают AppendRevision
```

Изменения в существующих хэндлерах:
- `POST /api/trackers/:id/entries` → теперь возвращает запись + автоматически делает `AppendRevision('create')`.
- `PATCH /api/entries/:id` → AppendRevision('update') с НОВЫМ состоянием.
- `DELETE /api/entries/:id` → soft-delete + AppendRevision('delete').

Новые хэндлеры:
- `GET /api/entries/:id/revisions` → `[{id, change_type, changed_at, changed_by, data, occurred_at, child_id, is_deleted}]`.
- `POST /api/entries/:id/restore/:revision_id` →
  1. Загружаем ревизию.
  2. UPDATE entries SET data_json=rev.data_json, occurred_at=rev.occurred_at, child_id=rev.child_id, is_deleted=rev.is_deleted, updated_at=now() WHERE id=:id.
  3. AppendRevision('restore').
  4. Возвращаем обновлённую запись.

В одной SQL-транзакции.

## Frontend — что добавляется

```
frontend/src/
├── api/
│   └── revisions.ts
├── views/
│   └── EntryHistory.vue       # отдельная страница или таб в edit-drawer
├── components/
│   └── trackers/
│       ├── EntryEditDrawer.vue   # рефакторинг Phase 4: внутри табы Current / History
│       └── RevisionRow.vue       # одна строка ревизии с действием Restore
└── lib/
    └── revisionSummary.ts     # короткая текстовка изменений: "amount: 120 → 150"
```

`revisionSummary.ts` сравнивает `revision.data` с предыдущей ревизией того же entry; в простом случае выводит «changed: amount, note», в первой — «Created». Для restore — «Restored to revision #X».

В списке entries (Phase 4):
- Toggle «Show deleted» (по умолчанию off).
- Удалённая запись отображается с иконкой trash и кнопкой `Restore` (вызывает `restore-deleted`).

## API эндпоинты

| Метод | Путь | Доступ | Описание |
|---|---|---|---|
| GET | `/api/entries/:id/revisions` | user | список ревизий, отсортированы desc по changed_at |
| POST | `/api/entries/:id/restore/:revision_id` | user | откатить к ревизии |
| POST | `/api/entries/:id/restore-deleted` | user | снять soft-delete |

## Зависимости (libs)

Ничего нового на бэке.
На фронте — ничего нового кроме shadcn-vue `tabs`.

## Acceptance criteria

- [ ] Создание новой entry создаёт ревизию `create`.
- [ ] Update создаёт ревизию `update` с новым состоянием.
- [ ] Soft-delete создаёт ревизию `delete` с `is_deleted=1`.
- [ ] Restore-revision создаёт новую ревизию `restore` (не удаляет существующие — append-only).
- [ ] История показывает все ревизии в обратном хронологическом порядке.
- [ ] Можно откатить к любой ревизии: данные применяются, в истории появляется новая запись `restore`.
- [ ] Удалённую запись можно «вернуть» из списка — она снова видна, в истории `restore` ревизия.
- [ ] Backfill при миграции: для каждой существующей entry создан ровно один `create` revision.
- [ ] Каскадное удаление трекера/entry удаляет revisions (через FK ON DELETE CASCADE).

## Verification

1. Создать запись Feeding (amount=120). История показывает 1 ревизию `create`.
2. Изменить amount=150, note="OK". История: 2 ревизии (create, update).
3. Откатить к первой ревизии. Текущее значение `amount=120, note=null`. История: 3 ревизии (create, update, restore).
4. Удалить запись. Toggle «Show deleted» → запись с trash-иконкой. История: 4 ревизии.
5. Restore-deleted. Запись снова в основном списке. История: 5 ревизий.
6. SQL: `SELECT change_type, changed_at FROM entry_revisions WHERE entry_id=1 ORDER BY id;` — порядок и типы как ожидается.
7. Откат с уже-existing entry, у которой нет полей из revision (например удалили поле из schema трекера) — UI показывает только пересекающиеся поля без падений.
