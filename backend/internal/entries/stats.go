package entries

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/bublya/baby-tracker/backend/internal/httpx"
	"github.com/bublya/baby-tracker/backend/internal/trackers"
	"github.com/go-chi/chi/v5"
)


const (
	statsBucketDay   = "day"
	statsBucketWeek  = "week"
	statsBucketMonth = "month"

	statsMaxRangeDays = 365 * 5
	statsDefaultDays  = 30
)

// ListInRangeForTracker returns non-deleted entries for one tracker in
// [from, to). Sorted ASC by occurred_at so callers can stream-aggregate.
//
// Profile scope is implicit via tracker_id (each tracker belongs to one profile).
func (s *Store) ListInRangeForTracker(ctx context.Context, trackerID int64, from, to time.Time) ([]*Entry, error) {
	q := `SELECT id, tracker_id, profile_id, data_json, occurred_at, created_by, is_deleted, created_at, updated_at
	        FROM entries
	       WHERE tracker_id=? AND is_deleted=0 AND occurred_at>=? AND occurred_at<?
	       ORDER BY occurred_at ASC, id ASC`
	rows, err := s.db.Conn().QueryContext(ctx, q, trackerID,
		from.UTC().Format(time.RFC3339), to.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Entry{}
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

type statsFieldEnvelope struct {
	Key  string `json:"key"`
	Type string `json:"type"`

	// number / duration
	Unit  string    `json:"unit,omitempty"`
	Sum   []float64 `json:"sum,omitempty"`
	Avg   []float64 `json:"avg,omitempty"`
	Min   []float64 `json:"min,omitempty"`
	Max   []float64 `json:"max,omitempty"`
	Count []int     `json:"count,omitempty"`

	// boolean
	TrueCount  []int `json:"true_count,omitempty"`
	FalseCount []int `json:"false_count,omitempty"`

	// select / multiselect
	Options []statsFieldOption `json:"options,omitempty"`
	ByValue map[string][]int   `json:"by_value,omitempty"`
}

type statsFieldOption struct {
	Value string            `json:"value"`
	Label map[string]string `json:"label"`
}

type statsResponse struct {
	TrackerID     int64                `json:"tracker_id"`
	From          string               `json:"from"`
	To            string               `json:"to"`
	Bucket        string               `json:"bucket"`
	Buckets       []string             `json:"buckets"`
	EntryCount    []int                `json:"entry_count"`
	HourHistogram []int                `json:"hour_histogram"`
	Fields        []statsFieldEnvelope `json:"fields"`
}

// Stats returns aggregated time-series for a tracker.
//
// Query params:
//
//	from   RFC3339 (optional, default to-30d)
//	to     RFC3339 (optional, default now)
//	bucket day|week|month (default day)
//
// Profile scope is implicit (each tracker belongs to one profile).
func (h *Handlers) Stats(w http.ResponseWriter, r *http.Request) {
	tid, err := strconv.ParseInt(chi.URLParam(r, "tid"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tracker id")
		return
	}
	q := r.URL.Query()

	bucket := q.Get("bucket")
	if bucket == "" {
		bucket = statsBucketDay
	}
	if bucket != statsBucketDay && bucket != statsBucketWeek && bucket != statsBucketMonth {
		httpx.WriteValidationError(w, "validation failed", map[string]string{
			"bucket": "must be one of: day, week, month",
		})
		return
	}

	now := time.Now().UTC()
	to := now
	if v := q.Get("to"); v != "" {
		t, err := parseFlexibleTime(v)
		if err != nil {
			httpx.WriteValidationError(w, "validation failed", map[string]string{
				"to": "must be ISO 8601 datetime",
			})
			return
		}
		to = t.UTC()
	}
	from := to.Add(-statsDefaultDays * 24 * time.Hour)
	if v := q.Get("from"); v != "" {
		t, err := parseFlexibleTime(v)
		if err != nil {
			httpx.WriteValidationError(w, "validation failed", map[string]string{
				"from": "must be ISO 8601 datetime",
			})
			return
		}
		from = t.UTC()
	}
	if !from.Before(to) {
		httpx.WriteValidationError(w, "validation failed", map[string]string{
			"from": "must be earlier than to",
		})
		return
	}
	if to.Sub(from) > time.Duration(statsMaxRangeDays)*24*time.Hour {
		httpx.WriteValidationError(w, "validation failed", map[string]string{
			"from": fmt.Sprintf("range must be ≤ %d days", statsMaxRangeDays),
		})
		return
	}

	t, err := h.Service.Trackers.GetByID(r.Context(), tid)
	if err != nil {
		writeError(w, http.StatusNotFound, "tracker not found")
		return
	}
	schema, err := trackers.ParseSchema(t.Schema)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "parse schema")
		return
	}

	entriesList, err := h.Service.Store.ListInRangeForTracker(r.Context(), tid, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load entries")
		return
	}

	bucketKeys := generateBucketKeys(from, to, bucket)
	bucketIdx := make(map[string]int, len(bucketKeys))
	for i, k := range bucketKeys {
		bucketIdx[k] = i
	}
	n := len(bucketKeys)

	resp := statsResponse{
		TrackerID:     tid,
		From:          from.Format(time.RFC3339),
		To:            to.Format(time.RFC3339),
		Bucket:        bucket,
		Buckets:       bucketKeys,
		EntryCount:    make([]int, n),
		HourHistogram: make([]int, 24),
	}

	type numericAcc struct {
		sum, min, max []float64
		count         []int
		seen          []bool
	}
	type boolAcc struct {
		t, f []int
	}
	type catAcc struct {
		byVal map[string][]int
	}

	type fieldAggKind int
	const (
		kindSkip fieldAggKind = iota
		kindNumeric
		kindBool
		kindSelect
		kindMultiselect
	)
	type fieldAgg struct {
		key   string
		kind  fieldAggKind
		field *trackers.FieldDef

		num *numericAcc
		bv  *boolAcc
		sel *catAcc
	}

	aggs := make([]*fieldAgg, 0, len(schema.Fields))
	for i := range schema.Fields {
		f := &schema.Fields[i]
		var kind fieldAggKind
		switch f.Type {
		case "number", "duration":
			kind = kindNumeric
		case "boolean":
			kind = kindBool
		case "select":
			kind = kindSelect
		case "multiselect":
			kind = kindMultiselect
		default:
			continue
		}
		a := &fieldAgg{key: f.Key, kind: kind, field: f}
		switch kind {
		case kindNumeric:
			a.num = &numericAcc{
				sum:   make([]float64, n),
				min:   make([]float64, n),
				max:   make([]float64, n),
				count: make([]int, n),
				seen:  make([]bool, n),
			}
		case kindBool:
			a.bv = &boolAcc{
				t: make([]int, n),
				f: make([]int, n),
			}
		case kindSelect, kindMultiselect:
			a.sel = &catAcc{byVal: map[string][]int{}}
			for _, opt := range f.Options {
				a.sel.byVal[opt.Value] = make([]int, n)
			}
		}
		aggs = append(aggs, a)
	}

	for _, e := range entriesList {
		occ, err := time.Parse(time.RFC3339, e.OccurredAt)
		if err != nil {
			continue
		}
		occ = occ.UTC()
		key := bucketKeyFor(occ, bucket)
		idx, ok := bucketIdx[key]
		if !ok {
			continue
		}
		resp.EntryCount[idx]++
		resp.HourHistogram[occ.Hour()]++

		var data map[string]any
		if len(e.Data) > 0 {
			_ = json.Unmarshal(e.Data, &data)
		}

		for _, a := range aggs {
			raw, present := data[a.key]
			if !present || raw == nil {
				continue
			}
			switch a.kind {
			case kindNumeric:
				v, ok := asFloat(raw)
				if !ok {
					continue
				}
				if !a.num.seen[idx] {
					a.num.min[idx] = v
					a.num.max[idx] = v
					a.num.seen[idx] = true
				} else {
					if v < a.num.min[idx] {
						a.num.min[idx] = v
					}
					if v > a.num.max[idx] {
						a.num.max[idx] = v
					}
				}
				a.num.sum[idx] += v
				a.num.count[idx]++
			case kindBool:
				b, ok := raw.(bool)
				if !ok {
					continue
				}
				if b {
					a.bv.t[idx]++
				} else {
					a.bv.f[idx]++
				}
			case kindSelect:
				s, ok := raw.(string)
				if !ok {
					continue
				}
				if arr, ok := a.sel.byVal[s]; ok {
					arr[idx]++
				}
			case kindMultiselect:
				arr, ok := raw.([]any)
				if !ok {
					continue
				}
				for _, item := range arr {
					s, ok := item.(string)
					if !ok {
						continue
					}
					if vals, ok := a.sel.byVal[s]; ok {
						vals[idx]++
					}
				}
			}
		}
	}

	for _, a := range aggs {
		env := statsFieldEnvelope{Key: a.key, Type: a.field.Type}
		switch a.kind {
		case kindNumeric:
			env.Unit = a.field.Unit
			env.Sum = a.num.sum
			env.Min = a.num.min
			env.Max = a.num.max
			env.Count = a.num.count
			env.Avg = make([]float64, n)
			for i := 0; i < n; i++ {
				if a.num.count[i] > 0 {
					env.Avg[i] = a.num.sum[i] / float64(a.num.count[i])
				}
				if !a.num.seen[i] {
					env.Min[i] = 0
					env.Max[i] = 0
				}
			}
		case kindBool:
			env.TrueCount = a.bv.t
			env.FalseCount = a.bv.f
		case kindSelect, kindMultiselect:
			env.ByValue = a.sel.byVal
			env.Options = make([]statsFieldOption, 0, len(a.field.Options))
			for _, opt := range a.field.Options {
				env.Options = append(env.Options, statsFieldOption{
					Value: opt.Value,
					Label: opt.Label,
				})
			}
		}
		resp.Fields = append(resp.Fields, env)
	}

	writeJSON(w, http.StatusOK, resp)
}

// generateBucketKeys returns aligned bucket keys (YYYY-MM-DD) covering
// [from, to) for the chosen bucket size.
func generateBucketKeys(from, to time.Time, bucket string) []string {
	keys := []string{}
	cur := alignBucket(from, bucket)
	for cur.Before(to) {
		keys = append(keys, cur.Format("2006-01-02"))
		cur = nextBucket(cur, bucket)
	}
	return keys
}

func alignBucket(t time.Time, bucket string) time.Time {
	t = t.UTC()
	switch bucket {
	case statsBucketDay:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	case statsBucketWeek:
		// ISO week start = Monday. time.Weekday: Sunday=0..Saturday=6.
		wd := int(t.Weekday())
		if wd == 0 {
			wd = 7
		}
		monday := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -(wd - 1))
		return monday
	case statsBucketMonth:
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	return t
}

func nextBucket(t time.Time, bucket string) time.Time {
	switch bucket {
	case statsBucketDay:
		return t.AddDate(0, 0, 1)
	case statsBucketWeek:
		return t.AddDate(0, 0, 7)
	case statsBucketMonth:
		return t.AddDate(0, 1, 0)
	}
	return t
}

func bucketKeyFor(t time.Time, bucket string) string {
	return alignBucket(t, bucket).Format("2006-01-02")
}

