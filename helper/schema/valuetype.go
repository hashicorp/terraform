package schema

//go:generate stringer -type=ValueType valuetype.go

import "fmt"

// ValueType is an enum of the type that can be represented by a schema.
type ValueType int

const (
	TypeInvalid ValueType = iota
	TypeBool
	TypeInt
	TypeFloat
	TypeString
	TypeList
	TypeMap
	TypeSet
	typeObject
)

// Zero returns the zero value for a type.
func (t ValueType) Zero() interface{} {
	switch t {
	case TypeInvalid:
		return nil
	case TypeBool:
		return false
	case TypeInt:
		return 0
	case TypeFloat:
		return 0.0
	case TypeString:
		return ""
	case TypeList:
		return []interface{}{}
	case TypeMap:
		return map[string]interface{}{}
	case TypeSet:
		return nil
	case typeObject:
		return map[string]interface{}{}
	default:
		panic(fmt.Sprintf("unknown type %s", t))
	}
}
