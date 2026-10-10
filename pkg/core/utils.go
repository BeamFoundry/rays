package core

import (
	"reflect"
	"sync"
	"unicode"
	"unicode/utf8"
	"unsafe"
)

type emptyInterface struct {
	tab  unsafe.Pointer
	data unsafe.Pointer
}

var initOnce sync.Once

var discoveredTypes map[string]reflect.Type

// typelinks returns a slice of the sections in each module, and a slice of *rtype offsets in each module. The types in each module are sorted by string.
//
//go:linkname typelinks reflect.typelinks
func typelinks() (sections []unsafe.Pointer, offset [][]int32)

// resolveTypeOff resolves an *rtype offset from a base type.
//
//go:linkname resolveTypeOff reflect.resolveTypeOff
func resolveTypeOff(rtype unsafe.Pointer, off int32) unsafe.Pointer

func loadTypes() {
	discoveredTypes = make(map[string]reflect.Type)
	var obj any = reflect.TypeFor[int]()
	sections, offset := typelinks()
	for i, offs := range offset {
		rodata := sections[i]
		for _, off := range offs {
			//nolint:gosec // Required
			(*emptyInterface)(unsafe.Pointer(&obj)).data = resolveTypeOff(rodata, off)
			typ := obj.(reflect.Type)
			if typ.Kind() == reflect.Pointer {
				if _, ok := discoveredTypes[typ.String()]; !ok {
					discoveredTypes[typ.String()] = typ
				}
				if _, ok := discoveredTypes[typ.Elem().String()]; !ok {
					discoveredTypes[typ.Elem().String()] = typ.Elem()
				}
			}
		}
	}
}

// FindTypesImplementingInterface() will return a slice of reflect.Type of types (pointers of structs)
// that implements one or more interfaces.
//
// Only exported types are considered.
func FindTypesImplementingInterface(iface reflect.Type, ifaces ...reflect.Type) []reflect.Type {
	initOnce.Do(loadTypes)
	implementingTypes := []reflect.Type{}

	for _, itype := range discoveredTypes {
		if itype.Kind() == reflect.Pointer && itype.Elem().Kind() == reflect.Struct /*&& unicode.IsUpper(r)*/ {
			if itype.Elem().Name() == "" {
				continue
			}
			b := []byte(itype.Elem().Name())
			if r, s := utf8.DecodeRune(b); s == 0 || !unicode.IsUpper(r) {
				continue
			}
			if implements := itype.Implements(iface); implements {
				for _, other := range ifaces {
					if !itype.Implements(other) {
						implements = false
					}
				}
				if implements {
					implementingTypes = append(implementingTypes, itype)
				}
			}
		}
	}
	return implementingTypes
}

// FindFieldWithAnnotation() will return the first field of a struct that is an anonymous interface
// and of type A.
//
// Contrary to [FindFieldForInterface], this search for a specific interface field,
// not one implementing that interface.
//
// It will return nil if no field can be found.
func FindFieldWithAnnotation[A any](reflectType reflect.Type) *reflect.StructField {
	annotation := reflect.TypeFor[A]()
	if reflectType.Kind() == reflect.Pointer && reflectType.Elem().Kind() == reflect.Struct {
		reflectType = reflectType.Elem()
	}

	for field := range reflectType.Fields() {
		if !field.Anonymous || field.Type.Kind() != reflect.Interface {
			continue
		}
		if field.Type == annotation {
			return &field
		}
	}
	return nil
}

// FindFieldForInterface() will return the first field of a struct that is an anonymous interface,
// implementing interface A.
//
// It will return nil if no field can be found.
func FindFieldForInterface[I any](reflectType reflect.Type) *reflect.StructField {
	iface := reflect.TypeFor[I]()
	if reflectType.Kind() == reflect.Pointer && reflectType.Elem().Kind() == reflect.Struct {
		reflectType = reflectType.Elem()
	}

	for field := range reflectType.Fields() {
		if !field.Anonymous || field.Type.Kind() != reflect.Interface {
			continue
		}
		if field.Type.Implements(iface) {
			return &field
		}
	}
	//nolint:gocritic // Keeping it for now
	/*
		for field := range reflectType.Fields() {
			if field.Anonymous && field.Type.Kind() == reflect.Struct {
				if f := FindFieldForInterface[I](reflect.PointerTo(field.Type)); f != nil {
					return f
				}
			}
		}
	*/
	return nil
}

// GetFieldFirstTag() will return the first tag value from a list of possible tags
// found on a struct field.
func GetFieldFirstTag(field *reflect.StructField, tags []string) string {
	for _, tag := range tags {
		if value, ok := field.Tag.Lookup(tag); ok {
			return value
		}
	}
	return ""
}

// UnprotectValue() let you get a new settable pointer to a [reflect.Value].
//
// It's main use if to let you write to a struct private field, like this framework do to inject
// [Beam] on private fields.
//
//nolint:gosec // Required
func UnprotectValue(value reflect.Value) reflect.Value {
	return reflect.NewAt(value.Type(), unsafe.Pointer(value.UnsafeAddr())).Elem()
}
