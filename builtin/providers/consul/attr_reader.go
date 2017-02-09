package consul

import "time"

type _AttrReader interface {
	GetBool(_SchemaAttr) bool
	GetBoolOK(_SchemaAttr) (b, ok bool)
	GetDurationOK(_SchemaAttr) (time.Duration, bool)
	GetFloat64OK(_SchemaAttr) (float64, bool)
	GetIntOK(_SchemaAttr) (int, bool)
	GetIntPtr(_SchemaAttr) *int
	GetString(_SchemaAttr) string
	GetStringOK(_SchemaAttr) (string, bool)
	GetStringPtr(_SchemaAttr) *string
	GetStringSlice(attrName _SchemaAttr) []string
	BackingType() string
}
