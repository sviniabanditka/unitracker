package trackers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
)

type Schema struct {
	Fields []FieldDef `json:"fields"`
}

type FieldDef struct {
	Key           string            `json:"key"`
	Label         map[string]string `json:"label"`
	Type          string            `json:"type"`
	Required      bool              `json:"required,omitempty"`
	IsPrimaryTime bool              `json:"isPrimaryTime,omitempty"`

	Unit    string        `json:"unit,omitempty"`
	Min     *float64      `json:"min,omitempty"`
	Max     *float64      `json:"max,omitempty"`
	Options []FieldOption `json:"options,omitempty"`
}

type FieldOption struct {
	Value string            `json:"value"`
	Label map[string]string `json:"label"`
}

var (
	keyRe = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

	allowedTypes = map[string]struct{}{
		"datetime":    {},
		"date":        {},
		"time":        {},
		"duration":    {},
		"number":      {},
		"text":        {},
		"longtext":    {},
		"select":      {},
		"multiselect": {},
		"boolean":     {},
		"color":       {},
	}

	timeTypes = map[string]struct{}{
		"datetime": {},
		"date":     {},
		"time":     {},
	}
)

// ParseSchema parses raw JSON, validates it, and returns the canonical form.
func ParseSchema(raw []byte) (*Schema, error) {
	if len(raw) == 0 {
		return nil, errors.New("schema_json is empty")
	}
	var s Schema
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("decode schema: %w", err)
	}
	if err := ValidateSchema(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

// MarshalSchema produces canonical JSON for storage.
func MarshalSchema(s *Schema) ([]byte, error) { return json.Marshal(s) }

func ValidateSchema(s *Schema) error {
	if s == nil {
		return errors.New("schema is nil")
	}
	if len(s.Fields) == 0 {
		return errors.New("schema must have at least one field")
	}
	seenKeys := map[string]struct{}{}
	primaryCount := 0
	for i := range s.Fields {
		f := &s.Fields[i]
		if !keyRe.MatchString(f.Key) {
			return fmt.Errorf("field[%d]: key %q must match [a-z][a-z0-9_]*", i, f.Key)
		}
		if _, dup := seenKeys[f.Key]; dup {
			return fmt.Errorf("field[%d]: duplicate key %q", i, f.Key)
		}
		seenKeys[f.Key] = struct{}{}
		if f.Label == nil || f.Label["en"] == "" {
			return fmt.Errorf("field[%d] (%s): label.en is required", i, f.Key)
		}
		if _, ok := allowedTypes[f.Type]; !ok {
			return fmt.Errorf("field[%d] (%s): unknown type %q", i, f.Key, f.Type)
		}
		if f.IsPrimaryTime {
			if _, ok := timeTypes[f.Type]; !ok {
				return fmt.Errorf("field[%d] (%s): isPrimaryTime requires type datetime|date|time", i, f.Key)
			}
			primaryCount++
		}
		switch f.Type {
		case "number":
			if f.Min != nil && f.Max != nil && *f.Min > *f.Max {
				return fmt.Errorf("field[%d] (%s): min > max", i, f.Key)
			}
		case "select", "multiselect":
			if len(f.Options) == 0 {
				return fmt.Errorf("field[%d] (%s): %s requires non-empty options", i, f.Key, f.Type)
			}
			seenVals := map[string]struct{}{}
			for j, opt := range f.Options {
				if opt.Value == "" {
					return fmt.Errorf("field[%d] (%s): option[%d] value is required", i, f.Key, j)
				}
				if _, dup := seenVals[opt.Value]; dup {
					return fmt.Errorf("field[%d] (%s): duplicate option value %q", i, f.Key, opt.Value)
				}
				seenVals[opt.Value] = struct{}{}
				if opt.Label == nil || opt.Label["en"] == "" {
					return fmt.Errorf("field[%d] (%s): option[%d] label.en is required", i, f.Key, j)
				}
			}
		}
		if f.Type != "number" && (f.Min != nil || f.Max != nil || f.Unit != "") {
			return fmt.Errorf("field[%d] (%s): min/max/unit only valid for number", i, f.Key)
		}
		if f.Type != "select" && f.Type != "multiselect" && len(f.Options) > 0 {
			return fmt.Errorf("field[%d] (%s): options only valid for select/multiselect", i, f.Key)
		}
	}
	if primaryCount > 1 {
		return errors.New("at most one field may have isPrimaryTime: true")
	}
	return nil
}

// PrimaryTimeKey returns the key of the field marked isPrimaryTime, or "" if none.
func (s *Schema) PrimaryTimeKey() string {
	if s == nil {
		return ""
	}
	for _, f := range s.Fields {
		if f.IsPrimaryTime {
			return f.Key
		}
	}
	return ""
}

