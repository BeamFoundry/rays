package lang

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"unicode"
	"unicode/utf8"
)

// Enum represents a type-safe enumeration.
//
// # Declaration
//
// To define an enum, declare an Enum type alias, an Enum struct and an
// initialized variable:
//
//	type HttpMethod = Enum[EnumHttpMethod]
//
//	type HttpMethodEnum struct {
//		GET,
//		HEAD,
//		POST,
//		PUT,
//		PATCH,
//		DELETE,
//		CONNECT,
//		OPTIONS,
//		TRACE,
//		USE HttpMethod
//	}
//
//	var HttpMethods = HttpMethod{}.New()
//
// To access the enum values:
//
//	var method HttpMethod = HttpMethods.GET
//
// # Custom string serialization
//
// Custom string serialization formats can be achieved by embedding an implementation of
// the [EnumStringer] interface.
// There is two implementations included : [TitleEnumStringer] or [LowerEnumStringer].
//
//	type Protocol = Enum[ProtocolEnum]
//
//	type ProtocolEnum struct {
//		LowerEnumStringer[ProtocolEnum]
//		HTTP,
//		HTTPS Protocol
//	}
//
//	var Protocols = Protocol{}.New()
//
//	Protocols.HTTPS.String() // will return "https".
//
// --
//
// To use zero-based ordinal values (e.g., 0, 1, 2) for serialization,
// add the `Enum:"Ordinal"` tag to the enum fields:
//
//	type IsSecure = Enum[IsSecureEnum]
//	type IsSecureEnum struct {
//		UNSECURE,
//		SECURE IsSecure `Enum:"Ordinal"`
//	}
//
// # Interfaces
//
// As seen [Enum] implements the [fmt.Stringer] to return a string. But [Enum] implements
// a few other interfaces :
//
//   - encoding.TextMarshaler & encoding.TextUnmarshaler : code/decode its name (or position if Ordinal) to []byte.
//   - json.Marshaler & json.Unmarshaler : code/decode it to/from JSON.
//   - sql.Scanner & driver.Valuer : to be used in a database model.
//
// Because Enum[T] implements [encoding.TextUnmarshaler], fields of custom enum types
// can be parsed directly in [github.com/BeamFoundry/rays/pkg/core.ConfigurationProperties].
type Enum[T comparable] struct {
	id, idx int
}

// New initializes and returns the enum container T, establishing the internal ordinals and names
// for all of its Enum[T] fields. It should be treated as a constructor and must be called before
// using any of the Enum[T] members or methods.
func (this Enum[T]) New() T {
	var t T

	reflectValue := reflect.ValueOf(&t).Elem()
	idx := 0
	for i := range reflectValue.NumField() {
		field := reflectValue.Field(i)
		if field.Type() != reflect.TypeFor[Enum[T]]() || reflectValue.Type().Field(i).Anonymous {
			continue
		}
		this.idx = idx
		idx++
		this.id = i

		reflectValue.Field(i).Set(reflect.ValueOf(this))
	}

	return t
}

// Compare compares this Enum to another Enum of the same type.
// It returns a negative integer, zero, or a positive integer as this Enum's
// ordinal is less than, equal to, or greater than the compared Enum's ordinal.
func (this Enum[T]) Compare(that Enum[T]) int {
	return this.Ordinal() - that.Ordinal()
}

// Ordinal returns the zero-based index of the enum value within the container T,
// excluding any fields that are not of type Enum[T].
func (this Enum[T]) Ordinal() int {
	return this.idx
}

// Name returns the raw field name of this enum value as defined in the container struct T.
func (this Enum[T]) Name() string {
	return reflect.TypeFor[T]().Field(this.id).Name
}

// From parses the given string and sets the enum to the matching value,
// returning an error if no match is found.
func (this *Enum[T]) From(text string) error {
	return this.UnmarshalText([]byte(text))
}

// String returns the string representation of the enum value. If the container struct T
// implements EnumStringer[T], it uses the customized format; otherwise, it returns the raw Name().
func (this Enum[T]) String() string {
	t := this.New()
	if stringer, ok := reflect.ValueOf(&t).Elem().Interface().(EnumStringer[T]); ok {
		return stringer.ToString(this)
	}
	return reflect.TypeFor[T]().Field(this.id).Name
}

// MarshalText implements encoding.TextMarshaler. It serializes the enum value to its
// string representation, or to its ordinal value if the enum is configured as ordinal.
func (this Enum[T]) MarshalText() ([]byte, error) {
	if this.isOrdinal() {
		return fmt.Appendf(nil, "%d", this.Ordinal()), nil
	}
	return fmt.Appendf(nil, `%s`, this.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler. It parses the enum value from its
// text representation, supporting both string names (customized or raw) and ordinal values.
func (this *Enum[T]) UnmarshalText(text []byte) error {
	t := this.New()

	reflectValue := reflect.ValueOf(&t).Elem()

	idx := 0
	prev := this.idx
	for i := range reflectValue.NumField() {
		field := reflectValue.Field(i)
		if field.Type() != reflect.TypeFor[Enum[T]]() || reflectValue.Type().Field(i).Anonymous {
			continue
		}

		this.idx = idx
		if this.isOrdinal() {
			if string(text) == strconv.Itoa(this.idx) {
				this.id = i
				return nil
			}
		} else if v, ok := field.Interface().(fmt.Stringer); ok && string(text) == v.String() {
			this.id = i
			return nil
		}
		idx++
	}
	this.idx = prev

	return NewError("IllegalArgumentException", "Invalid value for Enum type", "Enum", t, "value", text)
}

// MarshalJSON implements json.Marshaler. It serializes the enum value to JSON, enclosing
// string representations in quotes or outputting raw numbers for ordinal enums.
func (this Enum[T]) MarshalJSON() ([]byte, error) {
	if this.isOrdinal() {
		return fmt.Appendf(nil, "%d", this.Ordinal()), nil
	}
	return fmt.Appendf(nil, `"%s"`, this.String()), nil
}

// UnmarshalJSON implements json.Unmarshaler. It deserializes a JSON value into the enum value.
func (this *Enum[T]) UnmarshalJSON(data []byte) error {
	if this.isOrdinal() {
		return this.UnmarshalText(data)
	}
	s, err := strconv.Unquote(string(data))
	if err != nil {
		return fmt.Errorf("enum: %w", err)
	}
	return this.UnmarshalText([]byte(s))
}

func (this Enum[T]) isOrdinal() bool {
	reflectType := reflect.TypeFor[T]()
	for field := range reflectType.Fields() {
		if field.Type != reflect.TypeFor[Enum[T]]() || field.Anonymous {
			continue
		}
		if value, ok := field.Tag.Lookup("Enum"); ok {
			if value == "Ordinal" {
				return true
			}
		}
	}
	return false
}

// Scan implements sql.Scanner, allowing database values to be scanned directly into the enum.
// It supports scanning from int64 for ordinal enums and string for name-based enums.
func (this *Enum[T]) Scan(src any) error {
	if reflect.TypeOf(src).Kind() == reflect.Int64 && this.isOrdinal() {
		return this.UnmarshalText(fmt.Appendf(nil, "%d", src))
	} else if reflect.TypeOf(src).Kind() == reflect.String && !this.isOrdinal() {
		if s, ok := src.(string); ok {
			return this.UnmarshalText([]byte(s))
		}
	}
	return errors.New("unable to scan this Enum")
}

// Value implements driver.Valuer, returning the database value for this enum.
// It returns an int64 for ordinal enums and a string for name-based enums.
func (this Enum[T]) Value() (driver.Value, error) {
	if this.isOrdinal() {
		return int64(this.Ordinal()), nil
	}
	return this.String(), nil
}

// EnumStringer defines a custom formatting interface for enums.
// If the container struct T implements EnumStringer[T] (usually by embedding a struct
// that implements it, such as LowerEnumStringer or TitleEnumStringer), then String()
// will delegate to ToString to format the enum value name.
type EnumStringer[T comparable] interface {
	// ToString returns a custom string representation of the given enum value.
	ToString(Enum[T]) string
}

// LowerEnumStringer is an implementation of EnumStringer[T] to have lower case name returned
type LowerEnumStringer[T comparable] struct{}

func (this LowerEnumStringer[T]) ToString(value Enum[T]) string {
	return String(value.Name()).ToLowerCase().String()
}

// TitleEnumStringer is an implementation of EnumStringer[T] to have the title case name returned.
type TitleEnumStringer[T comparable] struct{}

func (this TitleEnumStringer[T]) ToString(value Enum[T]) string {
	name := value.Name()
	r, size := utf8.DecodeRuneInString(name)
	uc := unicode.ToUpper(r)
	return string(uc) + String(name[size:]).ToLowerCase().String()
}
