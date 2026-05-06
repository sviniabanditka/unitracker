# Phase 9 — Interactive Charts

## Цель

На странице каждого трекера (`/trackers/:id`) появляется секция Charts с интерактивными графиками: динамика во времени, гистограмма по часам, calendar-heatmap за год, отдельные карточки на каждое чартируемое поле схемы. Графики реагируют на смену активного ребёнка, локали, темы и выбранного диапазона/бакета.

## Объём (Scope)

- Бэкенд: единый `GET /api/trackers/{tid}/stats?from=&to=&bucket=&child_id=` возвращает плотные временные ряды (entries-count + per-field агрегации) и hour-of-day гистограмму одним ответом.
- Фронт-секция Charts в `TrackerView`:
  - Универсальные карточки: «Entries per bucket», «Hour of day», «Calendar heatmap (last year)».
  - На каждое чартируемое поле схемы — отдельная карточка:
    - `number`, `duration` — line/bar по sum/avg/min/max (toggle), для duration — форматтер ms→`H:MM:SS`.
    - `boolean` — stacked bar true/false.
    - `select`, `multiselect` — stacked bar по опциям с локализованными подписями (`resolveLabel`).
- Селекторы:
  - bucket: `day | week | month`.
  - range: `7d | 30d | 90d | 1y | custom` (две даты).
- Реактивность на: смену трекера, активного ребёнка из `ChildSwitcher`, локали EN/UK, тёмной/светлой темы.
- Lazy-load: `ChartsSection.vue` через `defineAsyncComponent`, чтобы echarts не ехал в основной бандл.

## Не входит в фазу

- Экспорт графиков (PNG/CSV).
- Сравнение детей в одной карточке (multi-child overlay).
- Прогнозы / тренд-линии / regression / moving averages.
- Drill-down с графика в конкретные entries.
- Серверное кэширование stats.

## Зависимости

- Phase 1–8.

## Изменения в БД

Нет.

## Backend — что добавляется / изменяется

```
backend/internal/entries/
├── stats.go               # ListInRangeForTracker + Stats handler + bucketing helpers
└── handlers.go            # MountTrackerScoped: + r.Get("/stats", h.Stats)
```

`Stats(w, r)`:
1. Парсит `tid` из path, query-параметры `from/to/bucket/child_id`. Дефолты: `bucket=day`, `to=now`, `from=to-30d`.
2. Тащит tracker (`Service.Trackers.GetByID`) и парсит схему (`trackers.ParseSchema`).
3. `Store.ListInRangeForTracker(ctx, tid, childID, from, to)` — один SELECT, не-удалённые.
4. Генерирует плотный список бакетов от `from` до `to`:
   - `day` — увеличиваем по `+24h`, ключ `2006-01-02`.
   - `week` — нормализуем в понедельник UTC, шаг `+7d`, ключ — дата понедельника.
   - `month` — 1-е число, шаг `AddDate(0,1,0)`, ключ — дата 1-го.
5. В одном проходе по entries:
   - находит индекс бакета по occurred_at;
   - инкрементит `entry_count[idx]`, `hour_histogram[hour]`;
   - для каждого чартируемого поля добавляет вклад в накопители: numeric → sum/min/max/count; boolean → true_count/false_count; select/multiselect → by_value[k][idx].
6. После прохода считает `avg = sum/count` (где `count > 0`).
7. Сериализует в `dashboardStats`-структуру и пишет ответ.

Validation:
- `bucket` ∉ {day,week,month} → 400 `validation.failed`.
- `child_id` не парсится как int → 400.
- `from > to` → 400.
- range `> 5y` → 400 (защита от огромных выборок).

Reuse:
- `internal/httpx/json.go` для error envelope.
- `internal/trackers.ParseSchema` для типов полей.
- `internal/entries/store.go` (приватные хелперы scanEntry, nullableInt и т.п.).

## Frontend — что добавляется / изменяется

```
frontend/src/
├── api/
│   └── stats.ts                       # statsApi.getTracker(id, params) + типы
├── lib/
│   ├── echarts.ts                     # tree-shaking регистрация компонентов
│   └── useTheme.ts                    # composable: ref<'light'|'dark'> с MutationObserver
└── components/trackers/charts/
    ├── ChartCard.vue                  # обёртка-карточка + empty state
    ├── ChartsSection.vue              # контейнер: селекторы + сетка карточек
    ├── EntryCountChart.vue            # bar entries × buckets
    ├── HourHistogramChart.vue         # bar 24×count
    ├── CalendarHeatmap.vue            # ECharts calendar series, отдельный 1y/day запрос
    ├── NumericFieldChart.vue          # line/bar number|duration, sum/avg/min/max toggle
    ├── BooleanFieldChart.vue          # stacked bar true/false
    └── CategoricalFieldChart.vue      # stacked bar по опциям
```

`views/TrackerView.vue` — `<ChartsSection v-if="tracker" :tracker="tracker" :schema="schema" :child-id="childId" />` между filters и entries-list секциями.

Локализация — `i18n/{en,uk}.json` блок `charts.*`:
- `title`, `entryCount`, `hourOfDay`, `calendarHeatmap`, `noData`.
- `bucket.{day,week,month}`, `range.{7d,30d,90d,1y,custom}`, `aggregation.{sum,avg,min,max,count}`.

## API эндпоинты

| Метод | Путь | Доступ | Описание |
|---|---|---|---|
| GET | `/api/trackers/{id}/stats` | user | агрегированные временные ряды + hour-histogram |

Response:
```json
{
  "tracker_id": 1,
  "from": "2026-04-04T00:00:00Z",
  "to": "2026-05-04T00:00:00Z",
  "bucket": "day",
  "buckets": ["2026-04-04", ...],
  "entry_count": [3, 5, 0, ...],
  "hour_histogram": [0,0,1,2,5,8,...],
  "fields": [
    {"key":"amount","type":"number","unit":"ml",
     "sum":[120,300,0,...],"avg":[60,75,0,...],
     "min":[60,50,0,...],"max":[60,100,0,...],"count":[2,4,0,...]},
    {"key":"type","type":"select",
     "options":[{"value":"bottle","label":{"en":"Bottle","uk":"Пляшка"}}, ...],
     "by_value":{"bottle":[2,3,0,...],"breast":[1,2,0,...]}},
    {"key":"wet","type":"boolean",
     "true_count":[2,3,0,...],"false_count":[1,2,0,...]}
  ]
}
```

## Зависимости (libs)

**Go**: ничего нового.

**JS**: `echarts@^5`, `vue-echarts@^7`. Минимально-возможный bundle через `echarts/core` и точечные `use()` нужных компонентов.

## Acceptance criteria

- [ ] `GET /api/trackers/:id/stats` возвращает плотные ряды одинаковой длины, корректные агрегации, hour_histogram длиной 24.
- [ ] Bucket day/week/month дают ожидаемое количество интервалов в окне.
- [ ] Невалидные параметры (`bucket`, `from > to`, диапазон > 5 лет, нечисловой `child_id`) → 400 с `code:"validation.failed"`.
- [ ] 401 без сессии, 404 для несуществующего tracker_id.
- [ ] На странице трекера секция Charts показывает: entry-count, hour-of-day, calendar heatmap, по карточке на каждое чартируемое поле.
- [ ] Селекторы bucket/range перерисовывают графики (запрос видно в DevTools).
- [ ] Активный child из `ChildSwitcher` фильтрует charts.
- [ ] Локаль EN↔UK переключает заголовки, подписи опций (`resolveLabel`), формат дат на оси (`Intl.DateTimeFormat(locale)`).
- [ ] Тёмная тема подхватывается реактивно (без перезагрузки).
- [ ] Mobile: карточки в один столбец, без горизонтальной прокрутки.
- [ ] Empty-state когда в окне нет данных.
- [ ] `go build && go vet` clean. `vue-tsc --noEmit && vite build` clean.
- [ ] echarts в отдельном lazy-чанке, не утяжеляет основной index.js.

## Verification

1. На свежей БД создать tracker «Feeding» с полями: `occurred_at`(datetime, isPrimaryTime), `amount`(number, unit `ml`, required), `type`(select [bottle, breast]), `note`(longtext).
2. Создать ~30 entries за последние 30 дней через API/UI.
3. `curl -s -b /tmp/cookies.txt 'http://localhost:8080/api/trackers/1/stats?bucket=day&from=2026-04-04T00:00:00Z&to=2026-05-04T00:00:00Z' | jq` — проверить длины массивов, что суммы по `amount.sum` совпадают с ручным подсчётом.
4. С `bucket=week` — 4–5 бакетов; ключи бакетов — понедельники.
5. `bucket=foo` → 400 с `code:"validation.failed"`.
6. Открыть `/trackers/1`, секция Charts видна; смена bucket/range — графики перерисовываются.
7. Tooltip на bar-карточке amount показывает `120 ml` (или `H:MM:SS` для duration-поля).
8. Calendar heatmap: hover на ячейку → дата + count.
9. Hour-of-day: 24 столбца, заголовок локализован.
10. Переключить локаль на UK — оси и подписи на украинском; опции select показываются по `label.uk`.
11. Переключить тёмную тему — графики читаемы, фон тёмный.
12. Resize на mobile (≤640px) — карточки складываются в один столбец.
13. На трекере без entries в выбранном окне — все карточки показывают «No data», ничего не падает.
14. Backend `go build ./... && go vet ./...` без ошибок. Frontend `vue-tsc -p tsconfig.app.json --noEmit && vite build` — clean. В выводе vite должен появиться отдельный chunk `ChartsSection-*.js`.
15. После всего — пометить Phase 9 в `docs/README.md` как `[x]`.
