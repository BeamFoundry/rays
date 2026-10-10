package config

import (
	"encoding"
	"reflect"
	"strconv"
	"time"

	. "github.com/BeamFoundry/rays/pkg/lang"
)

// Convert returns a reflect.Value from an "any" value cast according to a reflect.Type
func Convert(value any, t reflect.Type) (reflect.Value, Error) {
	if reflect.TypeOf(value).AssignableTo(t) {
		return reflect.ValueOf(value), nil
	}
	if reflect.TypeOf(value).Kind() == reflect.String {
		val, _ := value.(string)

		pointerTo := func(t reflect.Type) reflect.Type {
			if t.Kind() != reflect.Pointer {
				return reflect.PointerTo(t)
			}
			return t
		}
		valueOf := func(value any, err Error) (reflect.Value, Error) {
			return reflect.ValueOf(value), err
		}
		valueOfErr := func(value any, err error) (reflect.Value, Error) {
			if err != nil {
				return reflect.Value{}, NewErrorFrom(err)
			}
			return reflect.ValueOf(value), nil
		}

		if t == reflect.TypeFor[time.Duration]() {
			return valueOfErr(time.ParseDuration(val))
		}
		if t == reflect.TypeFor[[]byte]() {
			return reflect.ValueOf([]byte(val)), nil
		}
		if unmarshaler, ok := reflect.New(pointerTo(t).Elem()).Interface().(encoding.TextUnmarshaler); ok {
			if err := unmarshaler.UnmarshalText([]byte(val)); err != nil {
				return reflect.Value{}, NewErrorFrom(err)
			}
			v := reflect.ValueOf(unmarshaler)
			if t.Kind() == reflect.Pointer {
				return v, nil
			}
			return v.Elem(), nil
		}
		switch t.Kind() {
		case reflect.Bool:
			return valueOfErr(strconv.ParseBool(val))
		case reflect.Int:
			return valueOf(ConvertStringNumeric[int](val))
		case reflect.Int8:
			return valueOf(ConvertStringNumeric[int8](val))
		case reflect.Int16:
			return valueOf(ConvertStringNumeric[int16](val))
		case reflect.Int32:
			return valueOf(ConvertStringNumeric[int32](val))
		case reflect.Int64:
			return valueOf(ConvertStringNumeric[int64](val))
		case reflect.Uint:
			return valueOf(ConvertStringNumeric[uint](val))
		case reflect.Uint8:
			return valueOf(ConvertStringNumeric[uint8](val))
		case reflect.Uint16:
			return valueOf(ConvertStringNumeric[uint16](val))
		case reflect.Uint32:
			return valueOf(ConvertStringNumeric[uint32](val))
		case reflect.Uint64:
			return valueOf(ConvertStringNumeric[uint64](val))
		case reflect.Float32:
			return valueOf(ConvertStringNumeric[float32](val))
		case reflect.Float64:
			return valueOf(ConvertStringNumeric[float64](val))
		default:
			return reflect.Value{}, NewError("ConversionError", "Unable to convert from string")
		}
	} else if t.Kind() == reflect.String {
		switch reflect.TypeOf(value).Kind() {
		case reflect.Bool:
			return reflect.ValueOf(strconv.FormatBool(value.(bool))), nil
		case reflect.Int:
			return reflect.ValueOf(ConvertNumericString(value.(int))), nil
		case reflect.Int8:
			return reflect.ValueOf(ConvertNumericString(value.(int8))), nil
		case reflect.Int16:
			return reflect.ValueOf(ConvertNumericString(value.(int16))), nil
		case reflect.Int32:
			return reflect.ValueOf(ConvertNumericString(value.(int32))), nil
		case reflect.Int64:
			return reflect.ValueOf(ConvertNumericString(value.(int64))), nil
		case reflect.Uint:
			return reflect.ValueOf(ConvertNumericString(value.(uint))), nil
		case reflect.Uint8:
			return reflect.ValueOf(ConvertNumericString(value.(uint8))), nil
		case reflect.Uint16:
			return reflect.ValueOf(ConvertNumericString(value.(uint16))), nil
		case reflect.Uint32:
			return reflect.ValueOf(ConvertNumericString(value.(uint32))), nil
		case reflect.Uint64:
			return reflect.ValueOf(ConvertNumericString(value.(uint64))), nil
		case reflect.Float32:
			return reflect.ValueOf(ConvertNumericString(value.(float32))), nil
		case reflect.Float64:
			return reflect.ValueOf(ConvertNumericString(value.(float64))), nil
		default:
			return reflect.Value{}, NewError("ConversionError", "Unable to convert to string")
		}
	}
	return reflect.Value{}, NewError("ConversionError", "Not a valid Type Conversion")
}

func ConvertNumeric[T2, T1 IntegerConstraint | FloatConstraint](value T1) T2 {
	return T2(value)
}

func ConvertStringNumeric[T IntegerConstraint | FloatConstraint](value string) (T, Error) {
	t := reflect.TypeFor[T]()
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		integer, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return T(0), NewErrorFrom(err)
		}
		return T(integer), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		integer, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return T(0), NewErrorFrom(err)
		}
		return T(integer), nil
	default:
		integer, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return T(0), NewErrorFrom(err)
		}
		return T(integer), nil
	}
}

func ConvertNumericString[T IntegerConstraint | FloatConstraint](value T) string {
	t := reflect.TypeFor[T]()
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(int64(value), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(uint64(value), 10)
	case reflect.Float32:
		return strconv.FormatFloat(float64(value), 'g', -1, 32)
	default:
		return strconv.FormatFloat(float64(value), 'g', -1, 64)
	}
}
