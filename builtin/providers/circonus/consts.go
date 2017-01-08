package circonus

const (
	// Provider-level constants
	defaultCirconusTagCategory typeTagCategory = "author"
	defaultCirconusTagValue    typeTagValue    = "terraform"

	defaultWarnTags = 30
)

const (
	// circonus_metric.* resource attribute names
	metricIDAttr   schemaAttr = "id"
	metricNameAttr schemaAttr = "name"
	metricTypeAttr schemaAttr = "type"
	metricTagsAttr schemaAttr = "tags"
	metricUnitAttr schemaAttr = "unit"
)

// Consts and their close relative, Go pseudo-consts.

// validMetricTypes: See `_metric_type`: https://login.circonus.com/resources/api/calls/metric
var validMetricTypes = validStringValues{"numeric", "text"}
