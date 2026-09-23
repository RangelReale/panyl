package panyl

import (
	"slices"
	"strconv"
)

// MapValue is a helper for handling map[string]any
type MapValue map[string]any

func (m MapValue) HasValue(name string) bool {
	_, ok := m[name]
	return ok
}

func (m MapValue) HasValues(name ...string) bool {
	for _, n := range name {
		if !m.HasValue(n) {
			return false
		}
	}
	return true
}

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
			return int(vv)
		case uint8:
			return int(vv)
		case uint16:
			return int(vv)
		case uint32:
			return int(vv)
		case uint64:
			return int(vv)
		case float32:
			return int(vv)
		case float64:
			return int(vv)
		}
	}
	return 0
}

func (m MapValue) FloatValue(name string) float64 {
	v, ok := m[name]
	if ok {
		switch vv := v.(type) {
		case float32:
			return float64(vv)
		case float64:
			return float64(vv)
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
		}
	}
	return 0
}

func (m MapValue) BoolValue(name string) bool {
	v, ok := m[name]
	if ok {
		switch vv := v.(type) {
		case bool:
			return vv
		case int:
			return vv != 0
		case string:
			b, err := strconv.ParseBool(vv)
			if err == nil {
				return b
			}
		}
	}
	return false
}

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
			m[name] = append(vv, value)
			return
		case []any:
			m[name] = append(vv, value)
			return
		}
	}
	m[name] = []string{value}
}

// ListValueContains returns whether a list value contains the string, using the same rules as ListValue.
func (m MapValue) ListValueContains(name string, value string) bool {
	return slices.Contains(m.ListValue(name), value)
}
