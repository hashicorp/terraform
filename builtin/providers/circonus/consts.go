package circonus

const (
	// Provider-level constants

	// defaultAutoTag determines the default behavior of circonus.auto_tag.
	defaultAutoTag = false

	// When auto_tag is enabled, the default tag category and value will be set to
	// the following value unless overriden.
	defaultCirconusTag circonusTag = "author:terraform"

	// When hashing a Set, default to a buffer this size
	defaultHashBufSize = 512

	providerAPIURLAttr  = "api_url"
	providerAutoTagAttr = "auto_tag"
	providerKeyAttr     = "key"

	apiConsulCheckBlacklist    = "check_name_blacklist"
	apiConsulDatacenterAttr    = "dc"
	apiConsulNodeBlacklist     = "node_blacklist"
	apiConsulServiceBlacklist  = "service_blacklist"
	apiConsulStaleAttr         = "stale"
	checkConsulTokenHeader     = `X-Consul-Token`
	checkConsulV1NodePrefix    = "node"
	checkConsulV1Prefix        = "/v1/health"
	checkConsulV1ServicePrefix = "service"
	checkConsulV1StatePrefix   = "state"
	defaultCheckConsulHTTPAddr = "http://consul.service.consul"
	defaultCheckConsulPort     = "8500"

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

	defaultCheckHTTPTrapAsync = false

	defaultCheckCloudWatchVersion = "2010-08-01"

	defaultCollectorDetailAttrs = 10

	defaultGraphDatapoints = 8
	defaultGraphLineStyle  = "stepped"
	defaultGraphStyle      = "line"
	defaultGraphFunction   = "gauge"

	metricUnit       = ""
	metricUnitRegexp = `^.*$`

	defaultRuleSetLast       = "300s"
	defaultRuleSetMetricType = "numeric"
	defaultRuleSetRuleLen    = 4
	defaultAlertSeverity     = 1
	defaultRuleSetWindowFunc = "average"
	ruleSetAbsentMin         = "70s"
)

// Consts and their close relative, Go pseudo-consts.

// validMetricTypes: See `type`: https://login.circonus.com/resources/api/calls/check_bundle
var validMetricTypes = validStringValues{
	`caql`,
	`composite`,
	`histogram`,
	`numeric`,
	`text`,
}

// validAggregateFuncs: See `aggregate_function`: https://login.circonus.com/resources/api/calls/graph
var validAggregateFuncs = validStringValues{
	`none`,
	`min`,
	`max`,
	`sum`,
	`mean`,
	`geometric_mean`,
}

// validGraphLineStyles: See `line_style`: https://login.circonus.com/resources/api/calls/graph
var validGraphLineStyles = validStringValues{
	`stepped`,
	`interpolated`,
}

// validGraphStyles: See `style`: https://login.circonus.com/resources/api/calls/graph
var validGraphStyles = validStringValues{
	`area`,
	`line`,
}

// validAxisAttrs: See `line_style`: https://login.circonus.com/resources/api/calls/graph
var validAxisAttrs = validStringValues{
	`left`,
	`right`,
}

// validGraphFunctionValues: See `derive`: https://login.circonus.com/resources/api/calls/graph
var validGraphFunctionValues = validStringValues{
	`counter`,
	`derive`,
	`gauge`,
}

// validRuleSetWindowFuncs: See `derive` or `windowing_func`: https://login.circonus.com/resources/api/calls/rule_set
var validRuleSetWindowFuncs = validStringValues{
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
	ruleSetMetricTypeNumeric = "numeric"
	ruleSetMetricTypeText    = "text"
)

// validRuleSetMetricTypes: See `metric_type`: https://login.circonus.com/resources/api/calls/rule_set
var validRuleSetMetricTypes = validStringValues{
	ruleSetMetricTypeNumeric,
	ruleSetMetricTypeText,
}
