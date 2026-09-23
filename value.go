package panyl

import (
	"encoding/json"
	"math"
	"slices"
	"strconv"
)

// MapValue is a helper for handling map[string]any
type MapValue map[string]any

// HasValue returns whether the key exists.
func (m MapValue) HasValue(name string) bool {
	_, ok := m[name]
	return ok
}

// HasValues returns whether all the keys exist.
func (m MapValue) HasValues(name ...string) bool {
	for _, n := range name {
		if !m.HasValue(n) {
			return false
		}
	}
	return true
}

// StringValue returns a string value. Returns "" if not found or not a string.
func (m MapValue) StringValue(name string) string {
	v, ok := m[name]
	if ok {
		switch vv := v.(type) {
		case string:
			return vv
		}
	}
	return ""
}

// IntValue returns an integer value, converting from any integer or float type (floats are truncated).
// Unsigned values above math.MaxInt return math.MaxInt. Returns 0 if not found or not a number.
func (m MapValue) IntValue(name string) int {
	v, ok := m[name]
	if ok {
		switch vv := v.(type) {
		case int:
			return vv
		case int8:
			return int(vv)
		case int16:
			return int(vv)
		case int32:
			return int(vv)
		case int64:
			return int(vv)
		case uint:
			return int(min(vv, math.MaxInt))
		case uint8:
			return int(vv)
		case uint16:
			return int(vv)
		case uint32:
			return int(vv)
		case uint64:
			return int(min(vv, math.MaxInt))
		case float32:
			return int(vv)
		case float64:
			return int(vv)
		case json.Number:
			if i, err := vv.Int64(); err == nil {
				return int(i)
			}
			if f, err := vv.Float64(); err == nil {
				return int(f)
			}
		}
	}
	return 0
}

// FloatValue returns a float value, converting from any integer or float type.
// Returns 0 if not found or not a number.
func (m MapValue) FloatValue(name string) float64 {
	v, ok := m[name]
	if ok {
		switch vv := v.(type) {
		case float32:
			return float64(vv)
		case float64:
			return vv
		case int:
			return float64(vv)
		case int8:
			return float64(vv)
		case int16:
			return float64(vv)
		case int32:
			return float64(vv)
		case int64:
			return float64(vv)
		case uint:
			return float64(vv)
		case uint8:
			return float64(vv)
		case uint16:
			return float64(vv)
		case uint32:
			return float64(vv)
		case uint64:
			return float64(vv)
		case json.Number:
			if f, err := vv.Float64(); err == nil {
				return f
			}
		}
	}
	return 0
}

// BoolValue returns a bool value. Numbers are true if not zero, and strings are parsed with strconv.ParseBool.
// Returns false if not found or not convertible.
func (m MapValue) BoolValue(name string) bool {
	v, ok := m[name]
	if ok {
		switch vv := v.(type) {
		case bool:
			return vv
		case string:
			b, err := strconv.ParseBool(vv)
			if err == nil {
				return b
			}
		case float32, float64, json.Number:
			return m.FloatValue(name) != 0
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return m.IntValue(name) != 0
		}
	}
	return false
}

// MapValue returns a map value. Returns nil if not found or not a map.
func (m MapValue) MapValue(name string) MapValue {
	v, ok := m[name]
	if ok {
		switch vv := v.(type) {
		case MapValue:
			return vv
		case map[string]any:
			return vv
		}
	}
	return nil
}

// ListValue returns a list of strings. A single string value is returned as a one-element list, and non-string
// elements of a []any list are ignored.
func (m MapValue) ListValue(name string) []string {
	v, ok := m[name]
	if ok {
		switch vv := v.(type) {
		case string:
			return []string{vv}
		case []string:
			return vv
		case []any:
			var ret []string
			for _, item := range vv {
				if s, ok := item.(string); ok {
					ret = append(ret, s)
				}
			}
			return ret
		}
	}
	return nil
}

// ListValueAdd adds a string to a list value, if not already present.
func (m MapValue) ListValueAdd(name string, value string) {
	if m.ListValueContains(name, value) {
		return
	}
	v, ok := m[name]
	if ok {
		switch vv := v.(type) {
		case string:
			m[name] = []string{vv, value}
			return
		case []string:
			// clip so a slice shared with another item (like a clone) is never modified
			m[name] = append(slices.Clip(vv), value)
			return
		case []any:
			m[name] = append(slices.Clip(vv), value)
			return
		}
	}
	m[name] = []string{value}
}

// ListValueContains returns whether a list value contains the string, using the same rules as ListValue.
func (m MapValue) ListValueContains(name string, value string) bool {
	return slices.Contains(m.ListValue(name), value)
}
