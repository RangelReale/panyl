package util

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// https://stackoverflow.com/a/33925425/784175
var cleanAnsiEscapeRE = regexp.MustCompile(`(\x9B|\x1B\[)[0-?]*[ -\/]*[@-~]`)

// AnsiEscapeString removes ANSI escape sequences from a string, returning whether any was found.
// If none was found, the string is returned unchanged.
// The 8-bit CSI introducer is recognized both as the U+009B character and as a raw 0x9B byte.
func AnsiEscapeString(s string) (bool, string) {
	es := s
	if strings.IndexByte(s, 0x9B) >= 0 {
		es = rawCSIToRune(s)
	}
	count := 0
	ret := cleanAnsiEscapeRE.ReplaceAllStringFunc(es, func(s string) string {
		count++
		return ""
	})
	if count > 0 {
		return true, ret
	}
	return false, s
}

// DoAnsiEscapeString removes ANSI escape sequences from a string.
func DoAnsiEscapeString(s string) string {
	_, es := AnsiEscapeString(s)
	return es
}

// rawCSIToRune replaces raw 0x9B bytes that are not part of a valid UTF-8 sequence with the U+009B character,
// as the regexp works on characters, not bytes.
func rawCSIToRune(s string) string {
	var sb strings.Builder
	sb.Grow(len(s) + 1)
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 && s[i] == 0x9B {
			sb.WriteRune('\u009B')
		} else {
			sb.WriteString(s[i : i+size])
		}
		i += size
	}
	return sb.String()
}
