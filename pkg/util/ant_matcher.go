package util

import (
	"bytes"
	"regexp"
	"unicode/utf8"

	. "github.com/BeamFoundry/rays/pkg/lang"
)

// AntMatcher is an implementation for Ant-style path patterns.
//
// Standard Ant path matching URLs rules apply :
//
//   - ? matches one character
//   - * matches zero or more characters
//   - ** matches zero or more directories in a path
//
// Examples
//
//   - "com/t?st.html" — matches "com/test.html" but also "com/tast.html" or "com/txst.html"
//   - "com/*.html" — matches all .html files in the com directory
//   - "com/**/test.html" — matches all test.html files underneath the com path
//   - "com/beamfoundry/**/*.html" — matches all .html files underneath the org/beamfoundry path
//   - "org/**/services/bla.html" — matches "org/beamfoundry/services/bla.html" but also "org/beamfoundry/testing/services/bla.html" and "org/services/bla.html"
type AntMatcher struct {
	pattern *regexp.Regexp
	orig    string
}

// New() returns optionally and AntMatcher object pointer.
//
// Internally the string pattern is transformed into a [regexp.Regexp].
// In case the regexp compilation fails an [github.com/BeamFoundry/rays/pkg/lang.Error] will be returned.
func (this AntMatcher) New(pattern string) (*AntMatcher, Error) {
	this.orig = pattern
	b := []byte(pattern)
	var buf bytes.Buffer
	buf.WriteRune('^')
	for len(b) > 0 {
		r, size := utf8.DecodeRune(b)
		switch r {
		case '\\':
			buf.WriteRune(r)
			s, sz := utf8.DecodeRune(b[size:])
			buf.WriteRune(s)
			b = b[size+sz:]
			continue
		case '.':
			buf.WriteString("\\.")
		case '?':
			buf.WriteRune('.')
		case '*':
			if s, sz := utf8.DecodeRune(b[size:]); s == '*' {
				buf.WriteString(".*")
				b = b[size+sz:]
				if t, tz := utf8.DecodeRune(b); t == '/' {
					b = b[tz:]
				}
				continue
			}
			buf.WriteString("[^/]*?")
		case '^', '$':
			buf.WriteRune('\\')
			buf.WriteRune(r)
		default:
			buf.WriteRune(r)
		}
		b = b[size:]
	}
	buf.WriteRune('$')
	p, err := regexp.Compile(buf.String())
	if err != nil {
		return nil, NewPatternSyntaxError(err.Error())
	}

	this.pattern = p
	return &this, nil
}

// Match() returns whether the n byte slice match the pattern.
//
// TODO: With 1.27 use generic method instead of two methods.
func (this *AntMatcher) Match(n []byte) bool {
	return this.pattern.Match(n)
}

// MatchString() returns whether the s string match the pattern
//
// TODO: With 1.27 use generic method instead of two methods.
func (this *AntMatcher) MatchString(s string) bool {
	return this.pattern.MatchString(s)
}

// String() returns the original string passed to the constructor
func (this *AntMatcher) String() string {
	return this.orig
}
