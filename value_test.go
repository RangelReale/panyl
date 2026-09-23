package panyl

import (
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
