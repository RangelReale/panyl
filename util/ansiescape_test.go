package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnsiEscapeString(t *testing.T) {
	for _, test := range []struct {
		name     string
		s        string
		found    bool
		expected string
	}{
		{"none", "plain text", false, "plain text"},
		{"empty", "", false, ""},
		{"7-bit CSI", "\x1b[31mred\x1b[0m text", true, "red text"},
		{"8-bit CSI character", "\u009b31mred\u009b0m text", true, "red text"},
		{"8-bit CSI raw byte", "\x9b31mred\x9b0m text", true, "red text"},
		{"raw byte and unicode", "\x9b1mcafé\x1b[0m", true, "café"},
		{"raw byte not a sequence", "a\x9b", false, "a\x9b"},
	} {
		t.Run(test.name, func(t *testing.T) {
			found, es := AnsiEscapeString(test.s)
			assert.Equal(t, test.found, found)
			assert.Equal(t, test.expected, es)
			assert.Equal(t, test.expected, DoAnsiEscapeString(test.s))
		})
	}
}
