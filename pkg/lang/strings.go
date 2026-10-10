package lang

import (
	"bytes"
	"strings"
	"unicode"
)

func ToStrings[S ByteStringConstraint](s S) Strings {
	return Strings(string(s))
}

type Strings string

func (s Strings) ToString() String {
	return String(s)
}

func (s Strings) String() string {
	return string(s)
}

// Compare returns an integer comparing two strings lexicographically.
// The result will be 0 if a == b, -1 if a < b, and +1 if a > b.
//
// Compare is included only for symmetry with package bytes.
// It is usually clearer and always faster to use the built-in string comparison operators ==, <, >, and so on.
func (s Strings) Compare(a string) int {
	return strings.Compare(string(s), a)
}

// Contains reports whether substr is within s.
func (s Strings) Contains(substr string) bool {
	return strings.Contains(string(s), substr)
}

// ContainsAny reports whether any Unicode code points in chars are within s.
func (s Strings) ContainsAny(chars string) bool {
	return strings.ContainsAny(string(s), chars)
}

// ContainsFunc reports whether any Unicode code points r within s satisfy f(r).
func (s Strings) ContainsFunc(f func(rune) bool) bool {
	return strings.ContainsFunc(string(s), f)
}

// ContainsRune reports whether the Unicode code point r is within s.
func (s Strings) ContainsRune(r rune) bool {
	return strings.ContainsRune(string(s), r)
}

// Count counts the number of non-overlapping instances of substr in s.
// If substr is an empty string, Count returns 1 + the number of Unicode code points in s.
func (s Strings) Count(substr string) int {
	return strings.Count(string(s), substr)
}

// Cut slices s around the first instance of sep, returning the text before and after sep.
// The found result reports whether sep appears in s. If sep does not appear in s, cut returns s, "", false.
func (s Strings) Cut(sep string) (before, after string, found bool) {
	return strings.Cut(string(s), sep)
}

// CutPrefix returns s without the provided leading prefix string and reports whether it found the prefix.
// If s doesn't start with prefix, CutPrefix returns s, false. If prefix is the empty string, CutPrefix returns s, true.
func (s Strings) CutPrefix(prefix string) (after string, found bool) {
	return strings.CutPrefix(string(s), prefix)
}

// CutSuffix returns s without the provided ending suffix string and reports whether it found the suffix.
// If s doesn't end with suffix, CutSuffix returns s, false. If suffix is the empty string, CutSuffix returns s, true.
func (s Strings) CutSuffix(suffix string) (before string, found bool) {
	return strings.CutSuffix(string(s), suffix)
}

// EqualFold reports whether s and t, interpreted as UTF-8 strings, are equal under simple Unicode case-folding, which is a more general form of case-insensitivity.
func (s Strings) EqualFold(t string) bool {
	return strings.EqualFold(string(s), t)
}

// Fields splits the string s around each instance of one or more consecutive white space characters, as defined by unicode.IsSpace, returning a slice of substrings of s or an empty slice if s contains only white space.
func (s Strings) Fields() []string {
	return strings.Fields(string(s))
}

// FieldsFunc splits the string s at each run of Unicode code points c satisfying f(c) and returns an array of slices of s.
// If all code points in s satisfy f(c) or the string is empty, an empty slice is returned.
//
// FieldsFunc makes no guarantees about the order in which it calls f(c) and assumes that f always returns the same value for a given c.
func (s Strings) FieldsFunc(f func(rune) bool) []string {
	return strings.FieldsFunc(string(s), f)
}

// HasPrefix tests whether the string s begins with prefix.
func (s Strings) HasPrefix(prefix string) bool {
	return strings.HasPrefix(string(s), prefix)
}

// HasSuffix tests whether the string s ends with suffix.
func (s Strings) HasSuffix(suffix string) bool {
	return strings.HasSuffix(string(s), suffix)
}

// Index returns the index of the first instance of substr in s, or -1 if substr is not present in s.
func (s Strings) Index(substr string) int {
	return bytes.Index([]byte(s), []byte(substr))
}

// IndexAny returns the index of the first instance of any Unicode code point from chars in s, or -1 if no Unicode code point from chars is present in s.
func (s Strings) IndexAny(char string) int {
	return strings.IndexAny(string(s), char)
}

// IndexByte returns the index of the first instance of c in s, or -1 if c is not present in s.
func (s Strings) IndexByte(c byte) int {
	return strings.IndexByte(string(s), c)
}

// IndexFunc returns the index into s of the first Unicode code point satisfying f(c), or -1 if none do.
func (s Strings) IndexFunc(f func(rune) bool) int {
	return strings.IndexFunc(string(s), f)
}

// IndexRune returns the index of the first instance of the Unicode code point r, or -1 if rune is not present in s.
// If r is utf8.RuneError, it returns the first instance of any invalid UTF-8 byte sequence.
func (s Strings) IndexRune(r rune) int {
	return strings.IndexRune(string(s), r)
}

// LastIndex returns the index of the last instance of substr in s, or -1 if substr is not present in s.
func (s Strings) LastIndex(substr string) int {
	return strings.LastIndex(string(s), substr)
}

// LastIndexAny returns the index of the last instance of any Unicode code point from chars in s, or -1 if no Unicode code point from chars is present in s.
func (s Strings) LastIndexAny(chars string) int {
	return strings.LastIndexAny(string(s), chars)
}

// LastIndexByte returns the index of the last instance of c in s, or -1 if c is not present in s.
func (s Strings) LastIndexByte(c byte) int {
	return strings.LastIndexByte(string(s), c)
}

// LastIndexFunc returns the index into s of the last Unicode code point satisfying f(c), or -1 if none do.
func (s Strings) LastIndexFunc(f func(rune) bool) int {
	return strings.LastIndexFunc(string(s), f)
}

// Map returns a copy of the string s with all its characters modified according to the mapping function.
// If mapping returns a negative value, the character is dropped from the string with no replacement.
func (s Strings) Map(mapping func(rune) rune) string {
	return strings.Map(mapping, string(s))
}

// Repeat returns a new string consisting of count copies of the string s.
//
// It panics if count is negative or if the result of (len(s) * count) overflows.
func (s Strings) Repeat(count int) string {
	return strings.Repeat(string(s), count)
}

// Replace returns a copy of the string s with the first n non-overlapping instances of old replaced by new.
// If old is empty, it matches at the beginning of the string and after each UTF-8 sequence, yielding up to k+1 replacements for a k-rune string.
// If n < 0, there is no limit on the number of replacements.
func (s Strings) Replace(old, replacement string, n int) string {
	return strings.Replace(string(s), old, replacement, n)
}

// ReplaceAll returns a copy of the string s with all non-overlapping instances of old replaced by new.
// If old is empty, it matches at the beginning of the string and after each UTF-8 sequence, yielding up to k+1 replacements for a k-rune string.
func (s Strings) ReplaceAll(old, replacement string) string {
	return strings.ReplaceAll(string(s), old, replacement)
}

// Split slices s into all substrings separated by sep and returns a slice of the substrings between those separators.
//
// If s does not contain sep and sep is not empty, Split returns a slice of length 1 whose only element is s.
//
// If sep is empty, Split splits after each UTF-8 sequence. If both s and sep are empty, Split returns an empty slice.
//
// It is equivalent to SplitN with a count of -1.
//
// To split around the first instance of a separator, see Cut.
func (s Strings) Split(sep string) []string {
	return strings.Split(string(s), sep)
}

// SplitAfter slices s into all substrings after each instance of sep and returns a slice of those substrings.
//
// If s does not contain sep and sep is not empty, SplitAfter returns a slice of length 1 whose only element is s.
//
// If sep is empty, SplitAfter splits after each UTF-8 sequence. If both s and sep are empty, SplitAfter returns an empty slice.
//
// It is equivalent to SplitAfterN with a count of -1.
func (s Strings) SplitAfter(sep string) []string {
	return strings.SplitAfter(string(s), sep)
}

// SplitAfterN slices s into substrings after each instance of sep and returns a slice of those substrings.
func (s Strings) SplitAfterN(sep string, n int) []string {
	return strings.SplitAfterN(string(s), sep, n)
}

// SplitN slices s into substrings separated by sep and returns a slice of the substrings between those separators.
func (s Strings) SplitN(sep string, n int) []string {
	return strings.SplitN(string(s), sep, n)
}

// ToLower returns s with all Unicode letters mapped to their lower case.
func (s Strings) ToLower() string {
	return strings.ToLower(string(s))
}

// ToLowerSpecial returns a copy of the string s with all Unicode letters mapped to their lower case using the case mapping specified by c.
func (s Strings) ToLowerSpecial(c unicode.SpecialCase) string {
	return strings.ToLowerSpecial(c, string(s))
}

// ToUpper returns s with all Unicode letters mapped to their upper case.
func (s Strings) ToUpper() string {
	return strings.ToUpper(string(s))
}

// ToUpperSpecial returns a copy of the string s with all Unicode letters mapped to their upper case using the case mapping specified by c.
func (s Strings) ToUpperSpecial(c unicode.SpecialCase) string {
	return strings.ToUpperSpecial(c, string(s))
}

// ToValidUTF8 returns a copy of the string s with each run of invalid UTF-8 byte sequences replaced by the replacement string, which may be empty.
func (s Strings) ToValidUTF8(replacement string) string {
	return strings.ToValidUTF8(string(s), replacement)
}

// Trim returns a slice of the string s with all leading and trailing Unicode code points contained in cutset removed.
func (s Strings) Trim(cutset string) string {
	return strings.Trim(string(s), cutset)
}

// TrimFunc returns a slice of the string s with all leading and trailing Unicode code points c satisfying f(c) removed.
func (s Strings) TrimFunc(f func(rune) bool) string {
	return strings.TrimFunc(string(s), f)
}

// TrimLeft returns a slice of the string s with all leading Unicode code points contained in cutset removed.
func (s Strings) TrimLeft(cutset string) string {
	return strings.TrimLeft(string(s), cutset)
}

// TrimLeftFunc returns a slice of the string s with all leading Unicode code points c satisfying f(c) removed.
func (s Strings) TrimLeftFunc(f func(rune) bool) string {
	return strings.TrimLeftFunc(string(s), f)
}

// TrimPrefix returns s without the provided leading prefix string. If s doesn't start with prefix, s is returned unchanged.
func (s Strings) TrimPrefix(prefix string) string {
	return strings.TrimPrefix(string(s), prefix)
}

// TrimRight returns a slice of the string s, with all trailing Unicode code points contained in cutset removed.
func (s Strings) TrimRight(cutset string) string {
	return strings.TrimRight(string(s), cutset)
}

// TrimRightFunc returns a slice of the string s with all trailing Unicode code points c satisfying f(c) removed.
func (s Strings) TrimRightFunc(f func(rune) bool) string {
	return strings.TrimRightFunc(string(s), f)
}

// TrimSpace returns a slice of the string s, with all leading and trailing white space removed, as defined by Unicode.
func (s Strings) TrimSpace() string {
	return strings.TrimSpace(string(s))
}

// TrimSuffix returns s without the provided trailing suffix string. If s doesn't end with suffix, s is returned unchanged.
func (s Strings) TrimSuffix(suffix string) string {
	return strings.TrimSuffix(string(s), suffix)
}
