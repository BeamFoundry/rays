package utils

import (
	"unicode"
	"unicode/utf8"
)

func Uncapitalize(text string) string {
	r, size := utf8.DecodeRuneInString(text)
	if r == utf8.RuneError && size <= 1 {
		return text
	}
	return string(unicode.ToLower(r)) + text[size:]
}
