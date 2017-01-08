package circonus

const (
	// Provider-level constants

	// defaultAutoTag determines the default behavior of circonus.auto_tag.
	defaultAutoTag = false

	// When auto_tag is enabled, the default tag category and value will be set to
	// the following values unless overriden.
	defaultCirconusTagCategory typeTagCategory = "author"
	defaultCirconusTagValue    typeTagValue    = "terraform"

	// If there are more than this number of tags a warning will be issued.
	defaultWarnTags = 30

	providerAPIURLAttr  = "api_url"
	providerAutoTagAttr = "auto_tag"
	providerKeyAttr     = "key"
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
