package util

import (
	"bytes"
	"unicode"
	"unicode/utf8"

	. "github.com/BeamFoundry/rays/pkg/lang"
)

// CaseUtils is a type redefinition of "string"
//
// As such it can be constructed using the type name as a function.
//
// Example :
//
//	CaseUtils("test").UpperCase().String() // Returns "TEST"
type CaseUtils string

// String() Returns a string version of this CaseUtils
func (this CaseUtils) String() string {
	return string(this)
}

// ToString() Returns a [github.com/BeamFoundry/rays/pkg/lang.String] version of this CaseUtils
func (this CaseUtils) ToString() String {
	return String(this)
}

// Capitalize() will change the first letter to uppercase case. No other letters are changed.
func (this CaseUtils) Capitalize() CaseUtils {
	r, size := utf8.DecodeRuneInString(this.String())
	if r == utf8.RuneError && size <= 1 {
		return this
	}
	uc := unicode.ToUpper(r)
	return CaseUtils(string(uc) + this.String()[size:])
}

// Capitalize() will change the first letter to lowercase case. No other letters are changed.
func (this CaseUtils) Uncapitalize() CaseUtils {
	r, size := utf8.DecodeRuneInString(string(this))
	if r == utf8.RuneError && size <= 1 {
		return this
	}
	lc := unicode.ToLower(r)
	return CaseUtils(string(lc) + this.String()[size:])
}

// LowerCase() will convert all the characters to lower case
func (this CaseUtils) LowerCase() CaseUtils {
	return CaseUtils(this.ToString().Strings().ToLower())
}

// UpperCase() will convert all the characters to upper case
func (this CaseUtils) UpperCase() CaseUtils {
	return CaseUtils(this.ToString().Strings().ToUpper())
}

// SnakeCase() will convert this string to a snake case format
func (this CaseUtils) SnakeCase() CaseUtils {
	b := []byte(this)
	var buf bytes.Buffer
	var prev rune
	for len(b) > 0 {
		curr, size := utf8.DecodeRune(b)
		switch {
		case unicode.IsSpace(curr):
			prev = '_'
			buf.WriteRune(prev)
		case unicode.IsUpper(curr):
			prevExists := false
			if prev != 0 && prev != '_' {
				prevExists = true
			}
			if prevExists && !unicode.IsUpper(prev) {
				buf.WriteRune('_')
			} else if prevExists && len(b[size:]) > 0 {
				if next, _ := utf8.DecodeRune(b[size:]); unicode.IsLower(next) {
					buf.WriteRune('_')
				}
			}
			prev = curr
			curr = unicode.ToLower(curr)
			buf.WriteRune(unicode.ToLower(curr))
		default:
			prev = curr
			buf.WriteRune(curr)
		}
		b = b[size:]
	}
	return CaseUtils(buf.String())
}
