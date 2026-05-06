# Phase 6 — Backup & Snapshots

## Цель

Система автоматически создаёт снапшоты БД по расписанию и по запросу. Админ может скачать любой снапшот, удалить старые, и **откатить всю систему к выбранному снапшоту через UI без рестарта контейнера**. Перед откатом автоматически создаётся `pre-restore` снапшот, чтобы откатиться от отката.

## Объём (Scope)

- Таблица `snapshots`, миграция `0006_snapshots.sql`.
- Таблица `settings` (создаётся в этой фазе, инициализируется дефолтами), миграция `0007_settings.sql`.
- Модуль `internal/backup`:
  - `Service` с методами `Create(type, note)`, `List`, `Get`, `Delete`, `Download(io.Writer)`, `Restore(id)`, `RetentionApply()`.
  - `Scheduler` на `gocron`: одна джоба, перепланируется при изменении `backup_interval_hours`.
- `MaintenanceGate` в `internal/db/`:
  - `Acquire(write bool)` — блокируется если включён maintenance, возвращает release.
  - `EnterMaintenance()` — взводит флаг, ждёт завершения активных write-операций (через RWMutex).
  - `ExitMaintenance()`.
  - HTTP middleware: при maintenance отвечает 503 с `Retry-After`.
- Снапшот-файлы кладутся в `${DATA_DIR}/backups/snapshot-YYYYMMDDTHHMMSS-{type}.db`.
- Retention: при создании нового снапшота применяется `backup_retention_count` (дефолт 20), удаляются самые старые с типом `auto`. Manual и pre-restore не удаляются автоматически.
- UI:
  - `admin/BackupsAdmin.vue` — таблица снапшотов (filename, size, type, note, created_at, by) + действия:
    - `Create snapshot now` (POST manual).
    - На каждой строке: `Download`, `Restore`, `Delete`.
  - Модал подтверждения для restore: «This will replace ALL data. A pre-restore snapshot will be created automatically. Continue?».
  - В UI индикатор `Maintenance mode` (toast + блокирующий overlay) во время restore.

## Не входит в фазу

- Загрузка снапшотов извне («Restore from uploaded file») — **может быть добавлена опционально**, но базово не нужна.
- Шифрование снапшотов.
- S3/cloud sync (пользователь подключает свой sync на хосте).
- Export в SQL-dump формате.

## Зависимости

- Phase 1-5.

## Изменения в БД

`0006_snapshots.sql`:

```sql
-- +goose Up
CREATE TABLE snapshots (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  filename TEXT NOT NULL UNIQUE,
  size_bytes INTEGER NOT NULL,
  type TEXT NOT NULL CHECK (type IN ('auto','manual','pre-restore')),
  note TEXT,
  created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_snapshots_type_created ON snapshots(type, created_at DESC);

-- +goose Down
DROP TABLE snapshots;
```

`0007_settings.sql`:

```sql
-- +goose Up
CREATE TABLE settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO settings(key, value) VALUES
  ('backup_interval_hours', '6'),
  ('backup_retention_count', '20'),
  ('app_name', 'Baby Tracker'),
  ('default_locale', 'en');

-- +goose Down
DROP TABLE settings;
```

> Settings ползволяют не плодить миграции для каждой опции. Phase 7 строит UI поверх этой таблицы.

## Backend — что добавляется

```
backend/internal/
├── db/
│   └── maintenance.go         # MaintenanceGate, middleware
├── backup/
│   ├── service.go             # Create/List/Get/Delete/Download/Restore/RetentionApply
│   ├── scheduler.go           # Scheduler.Start, Reschedule(intervalHours)
│   └── handlers.go            # /api/admin/snapshots/*
└── settings/
    ├── store.go               # Get/Set/All; типизированные геттеры
    └── handlers.go            # /api/admin/settings (фактически наполняется в Phase 7)
```

### `MaintenanceGate` (упрощённая идея)

```go
type Gate struct {
    mu          sync.RWMutex   // защищает inMaintenance + waiting
    inMaint     atomic.Bool
    activeOps   sync.WaitGroup
}

// Любой write-эндпоинт оборачивается:
func (g *Gate) AcquireWrite() (release func(), err error) {
    if g.inMaint.Load() { return nil, ErrMaintenance }
    g.activeOps.Add(1)
    return func(){ g.activeOps.Done() }, nil
}

func (g *Gate) Enter(ctx context.Context) error {
    g.inMaint.Store(true)
    // wait active writes drain (с таймаутом)
    done := make(chan struct{})
    go func(){ g.activeOps.Wait(); close(done) }()
    select {
    case <-done: return nil
    case <-ctx.Done(): return ctx.Err()
    }
}

func (g *Gate) Exit() { g.inMaint.Store(false) }
```

HTTP middleware: когда `inMaint=true`, отвечать `503 Service Unavailable` с `Retry-After: 30` для всех `POST/PATCH/PUT/DELETE`. GET-ы можно временно тоже резать (проще), либо разрешать read-only — на этом этапе **режем все**, чтобы UI показал maintenance-плашку и не путался данными во время swap файла.

### `Service.Create(type, note, byUserID)`

1. Сгенерить filename `snapshot-{utc.Format("20060102T150405Z")}-{type}.db`.
2. Полный путь: `${DATA_DIR}/backups/{filename}`.
3. Выполнить `VACUUM INTO ?` с этим путём через тот же `*sql.DB` (на боевой БД, без maintenance — операция консистентна).
4. `os.Stat` → размер.
5. INSERT в `snapshots`.
6. Если `type='auto' || type='manual'` — вызвать `RetentionApply()` (только для `auto`).
7. Возвращаем созданный snapshot.

### `Service.Restore(id)`

1. Загрузить snapshot, проверить что файл существует.
2. **Сначала** `Service.Create('pre-restore', "before restoring snapshot #" + id, byUserID)`.
3. `gate.Enter(ctx)` (с таймаутом 30s).
4. Закрыть pool: вытащить `*sql.DB.Close()` через рефакторинг — на самом деле проще не закрывать, а:
   - Открыть source DB read-only.
   - На live `db` сделать `BEGIN IMMEDIATE; DELETE FROM ...; INSERT INTO ... SELECT FROM source.* (через ATTACH);` — это сложно из-за схемы.
   - **Простейший, и ок для self-hosted**: закрыть pool целиком, скопировать файл, открыть заново.
5. Реализация шага 4 простым способом:
   - В `internal/db` храним указатель на `*sql.DB` через **обёртку** `Database` с RWMutex и методом `Replace(newDB *sql.DB)`.
   - Все хэндлеры берут `db.Conn()` через эту обёртку (под RLock).
   - `Restore` берёт WLock → закрывает текущий → копирует файл (`io.Copy`) → открывает новый → ставит указатель.
6. После открытия — прогнать миграции goose (на случай если снапшот старее), хотя обычно noop.
7. `gate.Exit()`.
8. Возвращаем 200.

> Важное: все WAL-файлы (`-wal`, `-shm`) до закрытия должны быть «слиты» в основной файл. SQLite делает checkpoint при `PRAGMA wal_checkpoint(TRUNCATE)`; либо просто удалить `-wal/-shm` после `Close()` чтобы старая БД не задержалась. После замены файла создание нового pool откроет свежие.

### Scheduler

`gocron.NewScheduler` с одной джобой `Service.Create('auto', "scheduled", nil)`. При старте читаем `settings.backup_interval_hours`. Метод `Reschedule(h)` останавливает текущую джобу и добавляет новую.

В Phase 6 интеграция со settings минимальная (читаем при старте). Hot-reload — Phase 7.

## Frontend — что добавляется

```
frontend/src/
├── api/
│   ├── snapshots.ts
│   └── settings.ts            # минимальный get для maintenance check (если нужно)
├── stores/
│   └── snapshots.ts
├── views/admin/
│   └── BackupsAdmin.vue
├── components/
│   ├── MaintenanceOverlay.vue   # глобальный, показывается при 503 от write-эндпоинта или /api/health
│   └── ConfirmDialog.vue
```

Логика maintenance overlay:
- `api/client.ts` интерсептит ответы 503: ставит `useMaintenanceStore().on()`, начинает поллить `/api/health` каждые 3s. Когда снова 200 → `off()` → перезагружает текущие данные.
- Compose делает `<MaintenanceOverlay v-if="maintenance.on">`.

`BackupsAdmin.vue`:
- Таблица с пагинацией.
- Колонки: created_at, type (badge), filename, size (форматированный), by (username), note, actions.
- Создать manual snapshot: модал с optional note.
- Restore: модал с предупреждением + чекбокс `I understand`.
- Download: открывает `/api/admin/snapshots/:id/download` в новой вкладке.

## API эндпоинты

| Метод | Путь | Доступ | Тело / параметры |
|---|---|---|---|
| GET | `/api/admin/snapshots` | admin | `?type=&limit=&cursor=` |
| POST | `/api/admin/snapshots` | admin | `{note?: string}` создаёт `manual` |
| GET | `/api/admin/snapshots/:id/download` | admin | `Content-Disposition: attachment; filename=...` |
| POST | `/api/admin/snapshots/:id/restore` | admin | запускает restore-flow |
| DELETE | `/api/admin/snapshots/:id` | admin | удаляет файл и запись |
| GET | `/api/health` | public | возвращает `{status, maintenance:bool}` |

## Зависимости (libs)

**Go**: `github.com/go-co-op/gocron/v2`.

**JS**: ничего нового.

## Acceptance criteria

- [ ] При старте Scheduler планирует первую авто-джобу через `interval` часов от текущего времени.
- [ ] `POST /api/admin/snapshots` создаёт файл и запись.
- [ ] Файл консистентен: можно открыть `sqlite3 snapshot-*.db` и увидеть данные.
- [ ] Retention: при `backup_retention_count=3` после 4-го авто-снапшота старейший `auto` удалён, manual и pre-restore не тронуты.
- [ ] Restore меняет состояние БД на состояние снапшота.
- [ ] Перед restore автоматически создаётся `pre-restore` снапшот; в `snapshots` он виден.
- [ ] Во время restore UI показывает overlay; после завершения — overlay уходит, данные обновлены.
- [ ] Активный write во время `Enter` корректно завершается (или таймаут 30s).
- [ ] Удаление снапшота через UI удаляет файл с диска и строку из БД.
- [ ] Файл, существующий на диске но не в БД (или наоборот) не ломает list (логирование, но не 500).

## Verification

1. Заполнить данными несколько entries.
2. `POST /api/admin/snapshots` через UI с note «manual #1». В `data/backups/` появился файл.
3. Открыть `sqlite3 data/backups/snapshot-*-manual.db ".tables"` — все таблицы и данные есть.
4. Удалить пару entries. UI обновился.
5. В админке нажать `Restore` на manual #1 → подтвердить.
6. UI: maintenance overlay 1-3 секунды. Затем — overlay исчезает, удалённые entries вернулись.
7. В списке снапшотов видно новый `pre-restore` снапшот.
8. Поставить `backup_interval_hours=0.05` (3 минуты) через SQL update в `settings` (Phase 7 даст UI). Перезапустить контейнер — ждём 3 минуты — появляется первый `auto` снапшот.
9. Создать ещё 4 авто-снапшота (или подменить). При retention_count=3 убедиться, что старейший `auto` удаляется.
10. Скачать снапшот через UI — файл скачивается с правильным `Content-Disposition`.
11. Параллельно: пока restore в процессе, попытка POST entry от другого юзера → 503 + UI overlay.
12. Удалить снапшот → файл с диска тоже удаляется.
