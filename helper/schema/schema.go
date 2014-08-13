package schema

// ValueType is an enum of the type that can be represented by a schema.
type ValueType int

const (
	TypeInvalid ValueType = iota
	TypeBoolean
	TypeInt
	TypeString
	TypeList
)

// Schema is used to describe the structure of a value.
type Schema struct {
	// Type is the type of the value and must be one of the ValueType values.
	Type     ValueType

	// If one of these is set, then this item can come from the configuration.
	// Both cannot be set. If Optional is set, the value is optional. If
	// Required is set, the value is required.
	Optional bool
	Required bool

	// The fields below relate to diffs: if Computed is true, then the
	// result of this value is computed (unless specified by config).
	// If ForceNew is true
	Computed bool
	ForceNew bool

	// Elem must be either a *Schema or a *Resource only if the Type is
	// TypeList, and represents what the element type is. If it is *Schema,
	// the element type is just a simple value. If it is *Resource, the
	// element type is a complex structure, potentially with its own lifecycle.
	Elem interface{}
}
