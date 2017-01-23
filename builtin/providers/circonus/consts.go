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

	defaultCheckJSONMethod  = "GET"
	defaultCheckJSONPort    = "443"
	defaultCheckJSONVersion = "1.1"

	defaultCheckICMPPingAvailability = 100.0
	defaultCheckICMPPingCount        = 5
	defaultCheckICMPPingInterval     = "2s"

	defaultCheckCAQLTarget = "q._caql"

	defaultCheckHTTPCodeRegexp = `^200$`
	defaultCheckHTTPMethod     = "GET"
	defaultCheckHTTPVersion    = "1.1"

	defaultCheckCloudWatchVersion = "2010-08-01"

	defaultTriggerAbsentBuffer = 10.0
	defaultTriggerAfter        = "0m"
	defaultTriggerLast         = "300s"
	defaultTriggerMetricType   = "numeric"
	defaultTriggerRuleLen      = 4
	defaultTriggerSeverity     = 1
	defaultTriggerWindowFunc   = "average"
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

// _ValidTriggerWindowFuncs: See `derive` or `windowing_func`: https://login.circonus.com/resources/api/calls/rule_set
var _ValidTriggerWindowFuncs = _ValidStringValues{
	`average`,
	`stddev`,
	`derive`,
	`derive_stddev`,
	`counter`,
	`counter_stddev`,
	`derive_2`,
	`derive_2_stddev`,
	`counter_2`,
	`counter_2_stddev`,
}

const (
	// Supported circonus_trigger.metric_types.  See `metric_type`:
	// https://login.circonus.com/resources/api/calls/rule_set
	_TriggerMetricTypeNumeric = "numeric"
	_TriggerMetricTypeText    = "text"
)

// _ValidTriggerMetricTypes: See `metric_type`: https://login.circonus.com/resources/api/calls/rule_set
var _ValidTriggerMetricTypes = _ValidStringValues{
	_TriggerMetricTypeNumeric,
	_TriggerMetricTypeText,
}
