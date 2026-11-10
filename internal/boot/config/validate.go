package config

import (
	"reflect"
	"strings"

	. "github.com/BeamFoundry/rays/pkg/core"
	. "github.com/BeamFoundry/rays/pkg/lang"
	"github.com/go-playground/validator/v10"
)

var validation *validator.Validate

func init() {
	validation = validator.New()
	validation.RegisterTagNameFunc(func(field reflect.StructField) string {
		if private, ok := field.Tag.Lookup("private"); ok {
			return private
		}
		return field.Name
	})
}

func Validate(obj any) Error {
	val := reflect.ValueOf(obj)

	// 1. Extract the original struct name
	origType := val.Elem().Type()
	rootName, _, _ := strings.Cut(origType.Name(), "[")
	// Ensure the wrapper field is exported
	rootName = strings.ToUpper(rootName[:1]) + rootName[1:]

	// 2. Generate the deep proxy of the original struct
	proxiedValPtr := proxyValue(val)
	proxiedVal := proxiedValPtr.Elem()

	// 3. Construct the outer wrapper struct type
	wrapperField := reflect.StructField{
		Name: rootName,
		Type: proxiedVal.Type(),
		// We don't need any tags here; the validator automatically steps into exported struct fields
	}
	wrapperType := reflect.StructOf([]reflect.StructField{wrapperField})

	// 4. Instantiate the wrapper and inject our proxied struct into it
	wrapperVal := reflect.New(wrapperType).Elem()
	wrapperVal.Field(0).Set(proxiedVal)

	// 5. Validate the wrapper!
	// Step 5: Pass our fully-exported shadow struct to the validator!
	if err := validation.Struct(wrapperVal.Interface()); err != nil {
		return NewErrorFrom(err)
	}
	return nil
}

// proxyType recursively generates a new reflect.Type where all struct fields are capitalized.
func proxyType(t reflect.Type) reflect.Type {
	switch t.Kind() {
	case reflect.Pointer:
		return reflect.PointerTo(proxyType(t.Elem()))
	case reflect.Slice:
		return reflect.SliceOf(proxyType(t.Elem()))
	case reflect.Map:
		return reflect.MapOf(proxyType(t.Key()), proxyType(t.Elem()))
	case reflect.Struct:
		var fields []reflect.StructField
		for f := range t.Fields() {
			exportName := strings.ToUpper(f.Name[:1]) + f.Name[1:]
			tag := string(f.Tag)
			if tag != "" {
				tag += " "
			}
			tag = tag + `private:"` + f.Name + `"`
			fields = append(fields, reflect.StructField{
				Name: exportName,
				Type: proxyType(f.Type),
				Tag:  reflect.StructTag(tag), // Preserve validation tags
			})
		}
		return reflect.StructOf(fields)
	default:
		return t
	}
}

// proxyValue recursively traverses the data, extracts private fields using unsafe,
// and populates the new dynamically generated shadow types.
func proxyValue(v reflect.Value) reflect.Value {
	//nolint:gocritic // Untestable for now
	/*
		if !v.IsValid() {
			return v
		}
	*/

	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return reflect.Zero(proxyType(v.Type()))
		}
		elem := proxyValue(v.Elem())
		ptr := reflect.New(elem.Type())
		ptr.Elem().Set(elem)
		return ptr

	case reflect.Struct:
		if !v.CanAddr() {
			addrVal := reflect.New(v.Type())
			addrVal.Elem().Set(v) // .Set() safely copies private fields in Go!
			v = addrVal.Elem()
		}

		t := v.Type()
		pt := proxyType(t)
		pval := reflect.New(pt).Elem()

		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			fieldVal := v.Field(i)

			if !f.IsExported() {
				fieldVal = UnprotectValue(fieldVal)
			}

			// Recursively proxy the extracted field
			pval.Field(i).Set(proxyValue(fieldVal))
		}
		return pval

	case reflect.Slice:
		if v.IsNil() {
			return reflect.Zero(proxyType(v.Type()))
		}
		pt := proxyType(v.Type())
		pval := reflect.MakeSlice(pt, v.Len(), v.Cap())
		for i := 0; i < v.Len(); i++ {
			pval.Index(i).Set(proxyValue(v.Index(i)))
		}
		return pval

	case reflect.Map:
		if v.IsNil() {
			return reflect.Zero(proxyType(v.Type()))
		}
		pt := proxyType(v.Type())
		pval := reflect.MakeMap(pt)
		iter := v.MapRange()
		for iter.Next() {
			// this.Proxy both keys and values recursively
			pval.SetMapIndex(proxyValue(iter.Key()), proxyValue(iter.Value()))
		}
		return pval

	default:
		// Base case: primitives (strings, ints, etc.)
		return v
	}
}
