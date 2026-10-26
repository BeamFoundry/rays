package utils

import (
	"reflect"
)

// IsNil reports whether c is nil or holds a nil pointer, map, or slice.
func IsNil(c any) bool {
	if c == nil {
		return true
	}
	v := reflect.ValueOf(c)
	return (v.Kind() == reflect.Pointer || v.Kind() == reflect.Map || v.Kind() == reflect.Slice) && v.IsNil()
}
