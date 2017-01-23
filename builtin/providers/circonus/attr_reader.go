package circonus

import "time"

type _AttrReader interface {
	Context() *_ProviderContext
	GetBool(_SchemaAttr) bool
	GetBoolOK(_SchemaAttr) (b, ok bool)
	GetDurationOK(_SchemaAttr) (time.Duration, bool)
	GetFloat64OK(_SchemaAttr) (float64, bool)
	GetIntOK(_SchemaAttr) (int, bool)
	GetListOK(_SchemaAttr) (_InterfaceList, bool)
	GetMap(_SchemaAttr) _InterfaceMap
	GetSetAsListOK(_SchemaAttr) (_InterfaceList, bool)
	GetString(_SchemaAttr) string
	GetStringOK(_SchemaAttr) (string, bool)
	GetStringPtr(_SchemaAttr) *string
	GetStringSlice(attrName _SchemaAttr) []string
	GetTags(_SchemaAttr) _Tags
	BackingType() string
}
