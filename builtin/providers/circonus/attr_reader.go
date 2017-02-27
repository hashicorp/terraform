package circonus

import "time"

type attrReader interface {
	Context() *providerContext
	GetBool(schemaAttr) bool
	GetBoolOK(schemaAttr) (b, ok bool)
	GetDurationOK(schemaAttr) (time.Duration, bool)
	GetFloat64OK(schemaAttr) (float64, bool)
	GetIntOK(schemaAttr) (int, bool)
	GetIntPtr(schemaAttr) *int
	GetListOK(schemaAttr) (interfaceList, bool)
	GetMap(schemaAttr) interfaceMap
	GetSetAsListOK(schemaAttr) (interfaceList, bool)
	GetString(schemaAttr) string
	GetStringOK(schemaAttr) (string, bool)
	GetStringPtr(schemaAttr) *string
	GetStringSlice(attrName schemaAttr) []string
	GetTags(schemaAttr) circonusTags
	BackingType() string
}
