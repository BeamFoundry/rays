package lang

import (
	"fmt"
	"path/filepath"
	"regexp"
	"unicode/utf8"
)

type String string

// Implement Stringer interface to optain a string back from String
func (this String) String() string {
	return string(this)
}

// Returns a Strings object from this String
func (this String) Strings() Strings {
	return Strings(this)
}

// Basename returns the last element of path.
// Trailing path separators are removed before extracting the last element.
// If the path is empty, Base returns ".".
// If the path consists entirely of separators, Base returns a single separator.
func (this String) Basename() String {
	return String(filepath.Base(string(this)))
}

// CharAt returns the char value at the specified index.
// An index ranges from 0 to length() - 1.
// The first char value of the sequence is at index 0, the next at index 1, and so on, as for array indexing.
func (this String) CharAt(index int) (byte, Error) {
	switch {
	case index < 0:
		return byte(0), NewIndexOutOfBoundsError("index can not be negative.")
	case index >= len(this):
		return byte(0), NewIndexOutOfBoundsError("index is not less than the string length")
	default:
		return this[index], nil
	}
}

// CodePointAt returns the rune value at the specified index.
// An index ranges from 0 to length() - 1.
// The first char value of the sequence is at index 0, the next at index 1, and so on, as for array indexing.
func (this String) CodePointAt(index int) (rune, Error) {
	r := []rune(this)
	switch {
	case index < 0:
		return utf8.RuneError, NewIndexOutOfBoundsError("index can not be negative.")
	case index >= len(r):
		return utf8.RuneError, NewIndexOutOfBoundsError("index is not less than the string length")
	default:
		return r[index], nil
	}
}

// Compares two strings lexicographically. See [Strings.Compare]
func (this String) CompareTo(a string) int {
	return this.Strings().Compare(a)
}

// Concatenates the specified string to the end of this String.
func (this String) Concat(a string) String {
	return this + String(a)
}

// Returns true if and only if this String contains the specified string.
func (this String) Contains(a string) bool {
	return this.Strings().Contains(a)
}

// Returns a formatted String using this String as the format and the arguments.
func (this String) Format(args ...any) String {
	return String(fmt.Sprintf(this.String(), args...))
}

// Tests if this string ends with the specified suffix.
func (this String) EndsWith(suffix string) bool {
	return this.Strings().HasSuffix(suffix)
}

// Compares this string to the specified String.
func (this String) Equals(a string) bool {
	return this.CompareTo(a) == 0
}

// Compares this String to another String, ignoring case considerations. See [Strings.EqualFold].
func (this String) EqualsToIgnoreCase(a string) bool {
	return this.Strings().EqualFold(a)
}

// Convert this String to byte slice
func (this String) GetBytes() []byte {
	return []byte(this)
}

// Returns the index within this String of the first occurrence of the specified character.
func (this String) IndexOf(char rune) int {
	return Strings(this).IndexRune(char)
}

// Returns true if, and only if, its length is 0.
func (this String) IsEmpty() bool {
	return len(this) == 0
}

// Returns true if the string is empty or contains only white space codepoints, otherwise false.
func (this String) IsBlank() bool {
	if this.IsEmpty() {
		return true
	}

	if this.Strings().TrimSpace() == "" {
		return true
	}

	return false
}

// Returns the length of this String.
func (this String) Length() int {
	return len([]rune(this))
}

// Tells whether or not this String matches the given regular expression.
func (this String) Matches(regex string) (bool, Error) {
	matched, err := regexp.MatchString(regex, string(this))
	if err != nil {
		return matched, NewPatternSyntaxError(err.Error())
	}
	return matched, nil
}

//nolint:revive // flag-parameter
func (this String) RegionMatches(ignoreCase bool, toffset int, other string, ooffset, length int) bool {
	src := this.SubstringOffset(toffset, toffset+length)
	dst := String(other).SubstringOffset(ooffset, ooffset+length).String()

	if ignoreCase {
		return src.EqualsToIgnoreCase(dst)
	}
	return src.Equals(dst)
}

// Replaces each substring of this String that matches the literal target sequence with the specified literal replacement sequence.
func (this String) Replace(target, replacement string) String {
	return String(this.Strings().Replace(target, replacement, -1))
}

// Returns a string resulting from replacing all occurrences of oldChar in this string with newChar.
func (this String) ReplaceRune(oldChar, newChar rune) String {
	return String(this.Strings().Replace(string(oldChar), string(newChar), -1))
}

// Replaces each substring of this String that matches the given regular expression with the given replacement.
func (this String) ReplaceAll(pattern, replacement string) (String, Error) {
	reg, err := regexp.Compile(pattern)
	if err != nil {
		return this, NewPatternSyntaxError(err.Error())
	}
	return String(reg.ReplaceAllString(string(this), replacement)), nil
}

// Replaces the first substring of this String that matches the given regular expression with the given replacement.
func (this String) ReplaceFirst(pattern, replacement string) (String, Error) {
	if reg, err := regexp.Compile(pattern); err != nil {
		return this, NewPatternSyntaxError(err.Error())
	} else if found := reg.FindString(string(this)); found != "" {
		return String(this.Strings().Replace(found, replacement, 1)), nil
	}
	return this, nil
}

// Splits this String around matches of the given regular expression.
func (this String) Split(pattern string) ([]string, Error) {
	reg, err := regexp.Compile(pattern)
	if err != nil {
		return []string{}, NewPatternSyntaxError(err.Error())
	}
	return reg.Split(string(this), -1), nil
}

// Splits this String around matches of the given regular expression.
func (this String) SplitN(pattern string, limit int) ([]string, Error) {
	reg, err := regexp.Compile(pattern)
	if err != nil {
		return []string{}, NewPatternSyntaxError(err.Error())
	}
	switch {
	case limit > 0:
		return reg.Split(string(this), limit), nil
	case limit < 0:
		return reg.Split(string(this), -1), nil
	default:
		s := reg.Split(string(this), -1)
		return s[:len(s)-1], nil
	}
}

// Tests if this String starts with the specified prefix.
func (this String) StartsWith(prefix string) bool {
	return this.Strings().HasPrefix(prefix)
}

// Returns a String that is a substring of this string.
func (this String) Substring(start int) String {
	return this.SubstringOffset(start, len(this))
}

// Returns a string that is a substring of this string.
func (this String) SubstringOffset(start, end int) String {
	if this.IsEmpty() {
		return this
	}

	if start < 0 {
		start = len(this) + start
	}

	if end < 0 {
		end = len(this) + end
	}

	if end > this.Length() {
		end = this.Length()
	}

	if start > end {
		return ""
	}

	if start < 0 {
		start = 0
	}

	if end < 0 {
		end = 0
	}

	return this[start:end]
}

// ToCharArray() returns a slice of rune.
func (this String) ToCharArray() []rune {
	return []rune(this)
}

// Converts all of the characters in this String to lower case.
func (this String) ToLowerCase() String {
	return String(this.Strings().ToLower())
}

// Converts all of the characters in this String to upper case.
func (this String) ToUpperCase() String {
	return String(this.Strings().ToUpper())
}

// Returns a string whose value is this string, with all leading and trailing space removed,
// where space is defined as any character whose codepoint is less than or equal to 'U+0020' (the space character).
func (this String) Trim() String {
	return String(this.Strings().TrimFunc(func(r rune) bool { return r <= rune(' ') }))
}
