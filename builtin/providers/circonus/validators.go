package circonus

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/circonus-labs/circonus-gometrics/api"
)

var knownCheckTypes map[CheckType]struct{}

const (
	// Misc package constants
	defaultCheckTypeName  = "default"
	defaultNumHTTPHeaders = 3
)

var defaultCheckTypeConfigSize map[CheckType]int

func init() {
	// The values come from manually tallying up various options per check
	// type located at:
	// https://login.circonus.com/resources/api/calls/check_bundle
	defaultCheckTypeConfigSize = map[CheckType]int{
		defaultCheckTypeName: 8,
		"http":               16 + defaultNumHTTPHeaders,
		"json":               13 + defaultNumHTTPHeaders,
	}

	checkTypes := []string{
		"caql", "cim", "circonuswindowsagent", "circonuswindowsagent,nad",
		"collectd", "composite", "dcm", "dhcp", "dns", "elasticsearch",
		"external", "ganglia", "googleanalytics", "haproxy", "http",
		"http,apache", "httptrap", "imap", "jmx", "json", "json,couchdb",
		"json,mongodb", "json,nad", "json,riak", "ldap", "memcached",
		"munin", "mysql", "newrelic_rpm", "nginx", "nrpe", "ntp",
		"oracle", "ping_icmp", "pop3", "postgres", "redis", "resmon",
		"smtp", "snmp", "snmp,momentum", "sqlserver", "ssh2", "statsd",
		"tcp", "varnish", "keynote", "keynote_pulse", "cloudwatch",
		"ec_console", "mongodb",
	}

	knownCheckTypes = make(map[CheckType]struct{}, len(checkTypes))
	for _, k := range checkTypes {
		knownCheckTypes[CheckType(k)] = struct{}{}
	}
}

func validateAuthMethod(v interface{}, key string) (warnings []string, errors []error) {
	validAuthMethod := regexp.MustCompile(`^(?:Basic|Digest|Auto)$`)

	if !validAuthMethod.MatchString(v.(string)) {
		errors = append(errors, fmt.Errorf(`Invalid %s specified (%q).  Valid parameters are: "Basic", "Digest", and "Auto"`, checkConfigAuthUserAttr, v.(string)))
	}

	return warnings, errors
}

func validateAuthPassword(v interface{}, key string) (warnings []string, errors []error) {
	validAuthPassword := regexp.MustCompile(`^.*`)

	if !validAuthPassword.MatchString(v.(string)) {
		errors = append(errors, fmt.Errorf("Invalid %s specified (%q)", checkConfigAuthPasswordAttr, "<redacted>"))
	}

	return warnings, errors
}

func validateAuthUser(v interface{}, key string) (warnings []string, errors []error) {
	validAuthUser := regexp.MustCompile(`[^:]*`)

	if !validAuthUser.MatchString(v.(string)) {
		errors = append(errors, fmt.Errorf("Invalid %s specified (%q)", checkConfigAuthUserAttr, v.(string)))
	}

	return warnings, errors
}

func validateCAChain(v interface{}, key string) (warnings []string, errors []error) {
	validCAChain := regexp.MustCompile(`.+`)

	if !validCAChain.MatchString(v.(string)) {
		errors = append(errors, fmt.Errorf("Invalid %s specified (%q)", checkConfigCAChainAttr, v.(string)))
	}

	return warnings, errors
}

func validateCertificateFile(v interface{}, key string) (warnings []string, errors []error) {
	validCertificateFile := regexp.MustCompile(`.+`)

	if !validCertificateFile.MatchString(v.(string)) {
		errors = append(errors, fmt.Errorf("Invalid %s specified (%q)", checkConfigCertificateFileAttr, v.(string)))
	}

	return warnings, errors
}

func validateCheck(cb *api.CheckBundle) error {
	if cb.Timeout > float64(cb.Period) {
		return fmt.Errorf("Timeout (%f) can not exceed period (%d)", cb.Timeout, cb.Period)
	}

	return nil
}

func validateCheckType(v interface{}, key string) (warnings []string, errors []error) {
	if _, ok := knownCheckTypes[CheckType(v.(string))]; !ok {
		warnings = append(warnings, fmt.Sprintf("Possibly unsupported check type: %s", v.(string)))
	}

	return warnings, errors
}

func validateCiphers(v interface{}, key string) (warnings []string, errors []error) {
	validCiphers := regexp.MustCompile(`.+`)

	if !validCiphers.MatchString(v.(string)) {
		errors = append(errors, fmt.Errorf("Invalid %s specified (%q)", checkConfigCiphersAttr, v.(string)))
	}

	return warnings, errors
}

func validateHTTPHeaders(v interface{}, key string) (warnings []string, errors []error) {
	validHTTPHeader := regexp.MustCompile(`.+`)
	validHTTPValue := regexp.MustCompile(`.+`)

	headers := v.(map[string]interface{})
	for k, vRaw := range headers {
		if !validHTTPHeader.MatchString(k) {
			errors = append(errors, fmt.Errorf("Invalid HTTP Header specified: %q", k))
			continue
		}

		v := vRaw.(string)
		if !validHTTPValue.MatchString(v) {
			errors = append(errors, fmt.Errorf("Invalid value for HTTP Header %q specified: %q", k, v))
		}
	}

	return warnings, errors
}

func validateHTTPVersion(v interface{}, key string) (warnings []string, errors []error) {
	validHTTPVersion := regexp.MustCompile(`\d+\.\d+`)

	if !validHTTPVersion.MatchString(v.(string)) {
		errors = append(errors, fmt.Errorf("Invalid %s specified (%q)", checkConfigHTTPVersionAttr, v.(string)))
	}

	return warnings, errors
}

func validateKeyFile(v interface{}, key string) (warnings []string, errors []error) {
	validKeyFile := regexp.MustCompile(`.+`)

	if !validKeyFile.MatchString(v.(string)) {
		errors = append(errors, fmt.Errorf("Invalid %s specified (%q)", checkConfigKeyFileAttr, v.(string)))
	}

	return warnings, errors
}

func validateMethod(v interface{}, key string) (warnings []string, errors []error) {
	validMethod := regexp.MustCompile(`\S+`)

	if !validMethod.MatchString(v.(string)) {
		errors = append(errors, fmt.Errorf("Invalid %s specified (%q)", checkConfigMethodAttr, v.(string)))
	}

	return warnings, errors
}

func validateMetricLimit(v interface{}, key string) (warnings []string, errors []error) {
	limit := v.(int)
	switch {
	case limit < -1:
		errors = append(errors, fmt.Errorf("%s can not be less than -1 (%d)", checkMetricLimitAttr, limit))
	case limit == 0, limit >= 1:
		// no op
	}

	return warnings, errors
}

func validateMetricType(v interface{}, key string) (warnings []string, errors []error) {
	value := v.(string)
	switch value {
	case "caql", "composite", "histogram", "numeric", "text":
	default:
		errors = append(errors, fmt.Errorf("unsupported metric type %s", value))
	}

	return warnings, errors
}

func validatePeriod(v interface{}, key string) (warnings []string, errors []error) {
	const (
		minPeriod = 30
		maxPeriod = 300
	)

	switch period := v.(int); {
	case period < minPeriod:
		errors = append(errors, fmt.Errorf("%s can not be less than %d seconds (%d)", checkPeriodAttr, minPeriod, period))
	case period > maxPeriod:
		errors = append(errors, fmt.Errorf("%s can not be more than %d seconds (%d)", checkPeriodAttr, maxPeriod, period))
	}

	return warnings, errors
}

func validateReadLimit(v interface{}, key string) (warnings []string, errors []error) {
	limit := v.(int)
	if limit <= 0 {
		errors = append(errors, fmt.Errorf("%s can not be less than 0 (%d)", checkConfigReadLimitAttr, limit))
	}

	return warnings, errors
}

func validateRedirectLimit(v interface{}, key string) (warnings []string, errors []error) {
	redirect := v.(int)
	if redirect < 0 {
		errors = append(errors, fmt.Errorf("%s can not be less than 0 (%d)", checkConfigRedirectsAttr, redirect))
	}

	return warnings, errors
}

func validateTag(v interface{}, key string) (warnings []string, errors []error) {
	tag := v.(string)
	if !strings.ContainsRune(tag, ':') {
		errors = append(errors, fmt.Errorf("tag %q is missing a category", tag))
	}

	return warnings, errors
}

func validateTags(v interface{}) error {
	for i, tagRaw := range v.([]interface{}) {
		tag := tagRaw.(string)
		if !strings.ContainsRune(tag, ':') {
			return fmt.Errorf("tag %q at position %d in tag list is missing a category", tag, i+1)
		}
	}

	return nil
}

func validateTimeout(v interface{}, key string) (warnings []string, errors []error) {
	const (
		checkTimeoutMin float64 = 0.0
		checkTimeoutMax float64 = 300.0
	)

	timeout := v.(float64)
	switch {
	case timeout < checkTimeoutMin:
		errors = append(errors, fmt.Errorf("%s can not be less than %f (%f)", checkTimeoutAttr, checkTimeoutMin, timeout))
	case timeout > checkTimeoutMax:
		errors = append(errors, fmt.Errorf("%s can not be more than %f (%f)", checkTimeoutAttr, checkTimeoutMax, timeout))
	}

	return warnings, errors
}
