package circonus

type _AttrDescr string
type _AttrDescrs map[_SchemaAttr]_AttrDescr

type _MetricType string
type _SchemaAttr string

type _MetricID string
type _MetricName string

type _TagCategory string
type _TagValue string
type _Tags map[_TagCategory]_TagValue

type _Tag struct {
	Category _TagCategory
	Value    _TagValue
}

type _Unit string

type _ValidString string
type _ValidStringValues []_ValidString
