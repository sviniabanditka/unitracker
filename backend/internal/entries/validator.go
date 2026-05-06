package entries

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bublya/baby-tracker/backend/internal/trackers"
)

var (
	hexColor6Re = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	hexColor3Re = regexp.MustCompile(`^#[0-9a-fA-F]{3}$`)
	dateRe      = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	timeRe      = regexp.MustCompile(`^\d{2}:\d{2}(:\d{2})?$`)
	datetimeRe  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2}(\.\d+)?)?(Z|[+-]\d{2}:?\d{2})?$`)
)

const (
	textMaxLen     = 500
	longtextMaxLen = 10000
)

// ValidateData walks the schema and returns a list of human-readable errors.
// An empty slice means the data is valid.
func ValidateData(s *trackers.Schema, data map[string]any) []string {
	errs := []string{}
	if s == nil || len(s.Fields) == 0 {
		return errs
	}
	known := map[string]struct{}{}
	for _, f := range s.Fields {
		known[f.Key] = struct{}{}
	}
	for k := range data {
		if _, ok := known[k]; !ok {
			errs = append(errs, fmt.Sprintf("unknown field %q", k))
		}
	}
	for i := range s.Fields {
		f := &s.Fields[i]
		v, present := data[f.Key]
		if !present || isNil(v) || isEmptyValue(v) {
			if f.Required {
				errs = append(errs, fmt.Sprintf("%s: required", f.Key))
			}
			continue
		}
		if e := validateValue(f, v); e != "" {
			errs = append(errs, fmt.Sprintf("%s: %s", f.Key, e))
		}
	}
	return errs
}

func isNil(v any) bool { return v == nil }

func isEmptyValue(v any) bool {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x) == ""
	case []any:
		return len(x) == 0
	}
	return false
}

func validateValue(f *trackers.FieldDef, v any) string {
	switch f.Type {
	case "datetime":
		s, ok := v.(string)
		if !ok {
			return "must be a string"
		}
		if !datetimeRe.MatchString(s) {
			return "must be ISO 8601 datetime"
		}
	case "date":
		s, ok := v.(string)
		if !ok {
			return "must be a string"
		}
		if !dateRe.MatchString(s) {
			return "must be YYYY-MM-DD"
		}
	case "time":
		s, ok := v.(string)
		if !ok {
			return "must be a string"
		}
		if !timeRe.MatchString(s) {
			return "must be HH:MM[:SS]"
		}
	case "duration":
		n, ok := asFloat(v)
		if !ok {
			return "must be a number"
		}
		if n < 0 {
			return "must be ≥ 0"
		}
		if n != float64(int64(n)) {
			return "must be an integer (milliseconds)"
		}
	case "number":
		n, ok := asFloat(v)
		if !ok {
			return "must be a number"
		}
		if f.Min != nil && n < *f.Min {
			return fmt.Sprintf("must be ≥ %v", *f.Min)
		}
		if f.Max != nil && n > *f.Max {
			return fmt.Sprintf("must be ≤ %v", *f.Max)
		}
	case "text":
		s, ok := v.(string)
		if !ok {
			return "must be a string"
		}
		if len(s) > textMaxLen {
			return fmt.Sprintf("must be ≤ %d chars", textMaxLen)
		}
	case "longtext":
		s, ok := v.(string)
		if !ok {
			return "must be a string"
		}
		if len(s) > longtextMaxLen {
			return fmt.Sprintf("must be ≤ %d chars", longtextMaxLen)
		}
	case "select":
		s, ok := v.(string)
		if !ok {
			return "must be a string"
		}
		if !hasOption(f, s) {
			return "must be one of allowed options"
		}
	case "multiselect":
		arr, ok := v.([]any)
		if !ok {
			return "must be an array"
		}
		seen := map[string]struct{}{}
		for _, item := range arr {
			s, ok := item.(string)
			if !ok {
				return "items must be strings"
			}
			if !hasOption(f, s) {
				return fmt.Sprintf("contains invalid option %q", s)
			}
			if _, dup := seen[s]; dup {
				return fmt.Sprintf("duplicate option %q", s)
			}
			seen[s] = struct{}{}
		}
	case "boolean":
		if _, ok := v.(bool); !ok {
			return "must be boolean"
		}
	case "color":
		s, ok := v.(string)
		if !ok {
			return "must be a string"
		}
		if !hexColor3Re.MatchString(s) && !hexColor6Re.MatchString(s) {
			return "must be hex color (#RGB or #RRGGBB)"
		}
	default:
		return fmt.Sprintf("unsupported type %s", f.Type)
	}
	return ""
}

func asFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case json.Number:
		f, err := x.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	}
	return 0, false
}

func hasOption(f *trackers.FieldDef, val string) bool {
	for _, o := range f.Options {
		if o.Value == val {
			return true
		}
	}
	return false
}

// ExtractOccurredAt returns the canonical occurred_at for an entry by reading
// the field marked isPrimaryTime (if any). When the schema has no primary time
// or the value is missing, fallback is used.
func ExtractOccurredAt(s *trackers.Schema, data map[string]any, fallback time.Time) (time.Time, error) {
	key := s.PrimaryTimeKey()
	if key == "" {
		return fallback.UTC(), nil
	}
	raw, ok := data[key]
	if !ok || isNil(raw) {
		return fallback.UTC(), nil
	}
	str, ok := raw.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("primary time field %s must be a string", key)
	}
	str = strings.TrimSpace(str)
	if str == "" {
		return fallback.UTC(), nil
	}
	var f *trackers.FieldDef
	for i := range s.Fields {
		if s.Fields[i].Key == key {
			f = &s.Fields[i]
			break
		}
	}
	if f == nil {
		return fallback.UTC(), nil
	}
	switch f.Type {
	case "datetime":
		layouts := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05", "2006-01-02T15:04"}
		for _, l := range layouts {
			if t, err := time.Parse(l, str); err == nil {
				return t.UTC(), nil
			}
		}
		return time.Time{}, fmt.Errorf("invalid datetime: %s", str)
	case "date":
		t, err := time.Parse("2006-01-02", str)
		if err != nil {
			return time.Time{}, err
		}
		return t.UTC(), nil
	case "time":
		for _, l := range []string{"15:04:05", "15:04"} {
			if t, err := time.Parse(l, str); err == nil {
				today := fallback.UTC()
				return time.Date(today.Year(), today.Month(), today.Day(),
					t.Hour(), t.Minute(), t.Second(), 0, time.UTC), nil
			}
		}
		return time.Time{}, fmt.Errorf("invalid time: %s", str)
	}
	return fallback.UTC(), nil
}
