package lang

// ByteStringConstraint is a constraint that permits any byte slice or string.
// If future releases of Go add new predeclared byte slice or string types,
// this constraint will be modified to include them.
type ByteStringConstraint interface {
	~string | ~[]byte
}

// SignedConstraint is a constraint that permits any signed integer type.
// If future releases of Go add new predeclared signed integer types,
// this constraint will be modified to include them.
type SignedConstraint interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

// UnsignedConstraint is a constraint that permits any unsigned integer type.
// If future releases of Go add new predeclared unsigned integer types,
// this constraint will be modified to include them.
type UnsignedConstraint interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// IntegerConstraint is a constraint that permits any integer type.
// If future releases of Go add new predeclared integer types,
// this constraint will be modified to include them.
type IntegerConstraint interface {
	SignedConstraint | UnsignedConstraint
}

// FloatConstraint is a constraint that permits any floating-point type.
// If future releases of Go add new predeclared floating-point types,
// this constraint will be modified to include them.
type FloatConstraint interface {
	~float32 | ~float64
}

// ComplexConstraint is a constraint that permits any complex numeric type.
// If future releases of Go add new predeclared complex numeric types,
// this constraint will be modified to include them.
type ComplexConstraint interface {
	~complex64 | ~complex128
}

// OrderedConstraint is a constraint that permits any ordered type: any type
// that supports the operators < <= >= >.
// If future releases of Go add new ordered types,
// this constraint will be modified to include them.
type OrderedConstraint interface {
	IntegerConstraint | FloatConstraint | ~string
}
