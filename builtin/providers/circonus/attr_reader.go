package circonus

import "time"

type _AttrReader interface {
	Context() *_ProviderContext
	GetBool(_SchemaAttr) bool
	GetBoolOK(_SchemaAttr) (b, ok bool)
	GetDurationOK(_SchemaAttr) (time.Duration, bool)
	GetIntOK(_SchemaAttr) (int, bool)
	GetSetAsListOk(_SchemaAttr) (_InterfaceList, bool)
	GetString(_SchemaAttr) string
	GetStringOk(_SchemaAttr) (string, bool)
	GetStringPtr(_SchemaAttr) *string
	GetTags(_SchemaAttr) _Tags
	BackingType() string
}
