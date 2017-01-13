package circonus

const (
	// Provider-level constants

	// defaultAutoTag determines the default behavior of circonus.auto_tag.
	defaultAutoTag = false

	// When auto_tag is enabled, the default tag category and value will be set to
	// the following value unless overriden.
	defaultCirconusTag _Tag = "author:terraform"

	// When hashing a Set, default to a buffer this size
	defaultHashBufSize = 512

	// If there are more than this number of tags a warning will be issued.
	defaultWarnTags = 30

	providerAPIURLAttr  = "api_url"
	providerAutoTagAttr = "auto_tag"
	providerKeyAttr     = "key"
)

// Consts and their close relative, Go pseudo-consts.

// _ValidMetricTypes: See `type`: https://login.circonus.com/resources/api/calls/check_bundle
var _ValidMetricTypes = _ValidStringValues{
	`caql`,
	`composite`,
	`histogram`,
	`numeric`,
	`text`,
}
