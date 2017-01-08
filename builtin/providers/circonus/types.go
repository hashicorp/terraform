package circonus

type metricType string
type schemaAttr string

type typeMetricID string
type typeMetricName string

type typeTagCategory string
type typeTagValue string
type typeTags map[typeTagCategory]typeTagValue

type typeTag struct {
	Category typeTagCategory
	Value    typeTagValue
}

type typeUnit string

type validString string
type validStringValues []validString
