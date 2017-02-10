package consul

import "time"

type attrReader interface {
	GetBool(schemaAttr) bool
	GetBoolOK(schemaAttr) (b, ok bool)
	GetDurationOK(schemaAttr) (time.Duration, bool)
	GetFloat64OK(schemaAttr) (float64, bool)
	GetIntOK(schemaAttr) (int, bool)
	GetIntPtr(schemaAttr) *int
	GetString(schemaAttr) string
	GetStringOK(schemaAttr) (string, bool)
	GetStringPtr(schemaAttr) *string
	GetStringSlice(attrName schemaAttr) []string
	BackingType() string
}
