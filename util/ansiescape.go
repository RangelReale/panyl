package util

import (
	"regexp"
)

// https://stackoverflow.com/a/33925425/784175
var cleanAnsiEscapeRE = regexp.MustCompile(`(\x9B|\x1B\[)[0-?]*[ -\/]*[@-~]`)

func AnsiEscapeString(s string) (bool, string) {
	count := 0
	ret := cleanAnsiEscapeRE.ReplaceAllStringFunc(s, func(s string) string {
		count++
		return ""
	})
	if count > 0 {
		return true, ret
	}
	return false, ""
}

func DoAnsiEscapeString(s string) string {
	if ok, es := AnsiEscapeString(s); ok {
		return es
	}
	return s
}
