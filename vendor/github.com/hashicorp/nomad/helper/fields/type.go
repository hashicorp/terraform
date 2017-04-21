package fields

// FieldType is the enum of types that a field can be.
type FieldType uint

const (
	TypeInvalid FieldType = 0
	TypeString  FieldType = iota
	TypeInt
	TypeBool
	TypeMap
	TypeArray
)

func (t FieldType) String() string {
	switch t {
	case TypeString:
		return "string"
	case TypeInt:
		return "integer"
	case TypeBool:
		return "boolean"
	case TypeMap:
		return "map"
	case TypeArray:
		return "array"
	default:
		return "unknown type"
	}
}

func (t FieldType) Zero() interface{} {
	switch t {
	case TypeString:
		return ""
	case TypeInt:
		return 0
	case TypeBool:
		return false
	case TypeMap:
		return map[string]interface{}{}
	case TypeArray:
		return []interface{}{}
	default:
		panic("unknown type: " + t.String())
	}
}
