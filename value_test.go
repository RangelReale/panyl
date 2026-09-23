package panyl

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapValue_MapValue(t *testing.T) {
	m := MapValue{
		"named": MapValue{"a": 1},
		"plain": map[string]any{"b": 2},
		"other": "x",
	}
	assert.Equal(t, MapValue{"a": 1}, m.MapValue("named"))
	assert.Equal(t, MapValue{"b": 2}, m.MapValue("plain"))
	assert.Nil(t, m.MapValue("other"))
	assert.Nil(t, m.MapValue("missing"))
}

func TestMapValue_ListValue(t *testing.T) {
	m := MapValue{
		"string":  "a",
		"strings": []string{"a", "b"},
		"any":     []any{"a", 1, "b"},
	}
	assert.Equal(t, []string{"a"}, m.ListValue("string"))
	assert.Equal(t, []string{"a", "b"}, m.ListValue("strings"))
	assert.Equal(t, []string{"a", "b"}, m.ListValue("any"))
	assert.Nil(t, m.ListValue("missing"))
}

func TestMapValue_ListValueContains(t *testing.T) {
	m := MapValue{
		"string":  "a",
		"strings": []string{"a", "b"},
		"any":     []any{"a", "b"},
	}
	for _, name := range []string{"string", "strings", "any"} {
		assert.True(t, m.ListValueContains(name, "a"), name)
		assert.False(t, m.ListValueContains(name, "c"), name)
	}
	assert.False(t, m.ListValueContains("missing", "a"))
}

func TestMapValue_ListValueAdd(t *testing.T) {
	m := MapValue{
		"string":  "a",
		"strings": []string{"a"},
		"any":     []any{"a"},
	}
	for _, name := range []string{"string", "strings", "any", "missing"} {
		m.ListValueAdd(name, "a")
		m.ListValueAdd(name, "b")
		m.ListValueAdd(name, "b")
		assert.Equal(t, []string{"a", "b"}, m.ListValue(name), name)
	}
}

func TestMapValue_IntValue(t *testing.T) {
	m := MapValue{
		"int":      42,
		"uint64":   uint64(math.MaxUint64),
		"uint":     uint(math.MaxUint),
		"float":    3.9,
		"number":   json.Number("12"),
		"fnumber":  json.Number("1.5"),
		"string":   "12",
		"negative": int8(-3),
	}
	assert.Equal(t, 42, m.IntValue("int"))
	assert.Equal(t, math.MaxInt, m.IntValue("uint64"))
	assert.Equal(t, math.MaxInt, m.IntValue("uint"))
	assert.Equal(t, 3, m.IntValue("float"))
	assert.Equal(t, 12, m.IntValue("number"))
	assert.Equal(t, 1, m.IntValue("fnumber"))
	assert.Equal(t, 0, m.IntValue("string"))
	assert.Equal(t, -3, m.IntValue("negative"))
	assert.Equal(t, 0, m.IntValue("missing"))
}

func TestMapValue_BoolValue(t *testing.T) {
	for _, test := range []struct {
		value    any
		expected bool
	}{
		{true, true},
		{false, false},
		{"true", true},
		{"1", true},
		{"no", false},
		{1, true},
		{0, false},
		{int64(2), true},
		{uint8(0), false},
		{uint64(1), true},
		{0.5, true},
		{float32(0), false},
		{json.Number("1"), true},
		{json.Number("0"), false},
		{[]string{"a"}, false},
	} {
		assert.Equal(t, test.expected, MapValue{"v": test.value}.BoolValue("v"), "%T(%v)", test.value, test.value)
	}
}
