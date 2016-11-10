package fields

// FieldSchema is a basic schema to describe the format of a configuration field
type FieldSchema struct {
	Type        FieldType
	Default     interface{}
	Description string
	Required    bool
}

// DefaultOrZero returns the default value if it is set, or otherwise
// the zero value of the type.
func (s *FieldSchema) DefaultOrZero() interface{} {
	if s.Default != nil {
		return s.Default
	}

	return s.Type.Zero()
}
