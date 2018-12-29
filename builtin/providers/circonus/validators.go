package circonus

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/errwrap"
)

var knownCheckTypes map[circonusCheckType]struct{}
var knownContactMethods map[contactMethods]struct{}

var userContactMethods map[contactMethods]struct{}
var externalContactMethods map[contactMethods]struct{}
var supportedHTTPVersions = validStringValues{"0.9", "1.0", "1.1", "2.0"}
var supportedMetricClusterTypes = validStringValues{
	"average", "count", "counter", "counter2", "counter2_stddev",
	"counter_stddev", "derive", "derive2", "derive2_stddev", "derive_stddev",
	"histogram", "stddev", "text",
}

func init() {
	checkTypes := []circonusCheckType{
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

	knownCheckTypes = make(map[circonusCheckType]struct{}, len(checkTypes))
	for _, k := range checkTypes {
		knownCheckTypes[k] = struct{}{}
	}

	userMethods := []contactMethods{"email", "sms", "xmpp"}
	externalMethods := []contactMethods{"slack"}

	knownContactMethods = make(map[contactMethods]struct{}, len(externalContactMethods)+len(userContactMethods))

	externalContactMethods = make(map[contactMethods]struct{}, len(externalMethods))
	for _, k := range externalMethods {
		knownContactMethods[k] = struct{}{}
		externalContactMethods[k] = struct{}{}
	}

	userContactMethods = make(map[contactMethods]struct{}, len(userMethods))
	for _, k := range userMethods {
		knownContactMethods[k] = struct{}{}
		userContactMethods[k] = struct{}{}
	}
}

func validateCheckType(v interface{}, key string) (warnings []string, errors []error) {
	if _, ok := knownCheckTypes[circonusCheckType(v.(string))]; !ok {
		warnings = append(warnings, fmt.Sprintf("Possibly unsupported check type: %s", v.(string)))
	}

	return warnings, errors
}

func validateCheckCloudWatchDimmensions(v interface{}, key string) (warnings []string, errors []error) {
	validDimmensionName := regexp.MustCompile(`^[\S]+$`)
	validDimmensionValue := regexp.MustCompile(`^[\S]+$`)

	dimmensions := v.(map[string]interface{})
	for k, vRaw := range dimmensions {
		if !validDimmensionName.MatchString(k) {
			errors = append(errors, fmt.Errorf("Invalid CloudWatch Dimmension Name specified: %q", k))
			continue
		}

		v := vRaw.(string)
		if !validDimmensionValue.MatchString(v) {
			errors = append(errors, fmt.Errorf("Invalid value for CloudWatch Dimmension %q specified: %q", k, v))
		}
	}

	return warnings, errors
}

func validateContactGroup(cg *api.ContactGroup) error {
	for i := range cg.Reminders {
		if cg.Reminders[i] != 0 && cg.AggregationWindow > cg.Reminders[i] {
			return fmt.Errorf("severity %d reminder (%ds) is shorter than the aggregation window (%ds)", i+1, cg.Reminders[i], cg.AggregationWindow)
		}
	}

	for severityIndex := range cg.Escalations {
		switch {
		case cg.Escalations[severityIndex] == nil:
			continue
		case cg.Escalations[severityIndex].After > 0 && cg.Escalations[severityIndex].ContactGroupCID == "",
			cg.Escalations[severityIndex].After == 0 && cg.Escalations[severityIndex].ContactGroupCID != "":
			return fmt.Errorf("severity %d escalation requires both and %s and %s be set", severityIndex+1, contactEscalateToAttr, contactEscalateAfterAttr)
		}
	}

	return nil
}

func validateContactGroupCID(attrName schemaAttr) func(v interface{}, key string) (warnings []string, errors []error) {
	return func(v interface{}, key string) (warnings []string, errors []error) {
		validContactGroupCID := regexp.MustCompile(config.ContactGroupCIDRegex)

		if !validContactGroupCID.MatchString(v.(string)) {
			errors = append(errors, fmt.Errorf("Invalid %s specified (%q)", attrName, v.(string)))
		}

		return warnings, errors
	}
}

func validateDurationMin(attrName schemaAttr, minDuration string) func(v interface{}, key string) (warnings []string, errors []error) {
	var min time.Duration
	{
		var err error
		min, err = time.ParseDuration(minDuration)
		if err != nil {
			return func(interface{}, string) (warnings []string, errors []error) {
				errors = []error{errwrap.Wrapf(fmt.Sprintf("Invalid time +%q: {{err}}", minDuration), err)}
				return warnings, errors
			}
		}
	}

	return func(v interface{}, key string) (warnings []string, errors []error) {
		d, err := time.ParseDuration(v.(string))
		switch {
		case err != nil:
			errors = append(errors, errwrap.Wrapf(fmt.Sprintf("Invalid %s specified (%q): {{err}}", attrName, v.(string)), err))
		case d < min:
			errors = append(errors, fmt.Errorf("Invalid %s specified (%q): minimum value must be %s", attrName, v.(string), min))
		}

		return warnings, errors
	}
}

func validateDurationMax(attrName schemaAttr, maxDuration string) func(v interface{}, key string) (warnings []string, errors []error) {
	var max time.Duration
	{
		var err error
		max, err = time.ParseDuration(maxDuration)
		if err != nil {
			return func(interface{}, string) (warnings []string, errors []error) {
				errors = []error{errwrap.Wrapf(fmt.Sprintf("Invalid time +%q: {{err}}", maxDuration), err)}
				return warnings, errors
			}
		}
	}

	return func(v interface{}, key string) (warnings []string, errors []error) {
		d, err := time.ParseDuration(v.(string))
		switch {
		case err != nil:
			errors = append(errors, errwrap.Wrapf(fmt.Sprintf("Invalid %s specified (%q): {{err}}", attrName, v.(string)), err))
		case d > max:
			errors = append(errors, fmt.Errorf("Invalid %s specified (%q): maximum value must be less than or equal to %s", attrName, v.(string), max))
		}

		return warnings, errors
	}
}

func validateFloatMin(attrName schemaAttr, min float64) func(v interface{}, key string) (warnings []string, errors []error) {
	return func(v interface{}, key string) (warnings []string, errors []error) {
		if v.(float64) < min {
			errors = append(errors, fmt.Errorf("Invalid %s specified (%f): minimum value must be %f", attrName, v.(float64), min))
		}

		return warnings, errors
	}
}

func validateFloatMax(attrName schemaAttr, max float64) func(v interface{}, key string) (warnings []string, errors []error) {
	return func(v interface{}, key string) (warnings []string, errors []error) {
		if v.(float64) > max {
			errors = append(errors, fmt.Errorf("Invalid %s specified (%f): maximum value must be %f", attrName, v.(float64), max))
		}

		return warnings, errors
	}
}

// validateFuncs takes a list of functions and runs them in serial until either
// a warning or error is returned from the first validation function argument.
func validateFuncs(fns ...func(v interface{}, key string) (warnings []string, errors []error)) func(v interface{}, key string) (warnings []string, errors []error) {
	return func(v interface{}, key string) (warnings []string, errors []error) {
		for _, fn := range fns {
			warnings, errors = fn(v, key)
			if len(warnings) > 0 || len(errors) > 0 {
				break
			}
		}
		return warnings, errors
	}
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

func validateGraphAxisOptions(v interface{}, key string) (warnings []string, errors []error) {
	axisOptionsMap := v.(map[string]interface{})
	validOpts := map[schemaAttr]struct{}{
		graphAxisLogarithmicAttr: struct{}{},
		graphAxisMaxAttr:         struct{}{},
		graphAxisMinAttr:         struct{}{},
	}

	for k := range axisOptionsMap {
		if _, ok := validOpts[schemaAttr(k)]; !ok {
			errors = append(errors, fmt.Errorf("Invalid axis option specified: %q", k))
			continue
		}
	}

	return warnings, errors
}

func validateIntMin(attrName schemaAttr, min int) func(v interface{}, key string) (warnings []string, errors []error) {
	return func(v interface{}, key string) (warnings []string, errors []error) {
		if v.(int) < min {
			errors = append(errors, fmt.Errorf("Invalid %s specified (%d): minimum value must be %d", attrName, v.(int), min))
		}

		return warnings, errors
	}
}

func validateIntMax(attrName schemaAttr, max int) func(v interface{}, key string) (warnings []string, errors []error) {
	return func(v interface{}, key string) (warnings []string, errors []error) {
		if v.(int) > max {
			errors = append(errors, fmt.Errorf("Invalid %s specified (%d): maximum value must be %d", attrName, v.(int), max))
		}

		return warnings, errors
	}
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

func validateRegexp(attrName schemaAttr, reString string) func(v interface{}, key string) (warnings []string, errors []error) {
	re := regexp.MustCompile(reString)

	return func(v interface{}, key string) (warnings []string, errors []error) {
		if !re.MatchString(v.(string)) {
			errors = append(errors, fmt.Errorf("Invalid %s specified (%q): regexp failed to match string", attrName, v.(string)))
		}

		return warnings, errors
	}
}

func validateTag(v interface{}, key string) (warnings []string, errors []error) {
	tag := v.(string)
	if !strings.ContainsRune(tag, ':') {
		errors = append(errors, fmt.Errorf("tag %q is missing a category", tag))
	}

	return warnings, errors
}

func validateUserCID(attrName string) func(v interface{}, key string) (warnings []string, errors []error) {
	return func(v interface{}, key string) (warnings []string, errors []error) {
		valid := regexp.MustCompile(config.UserCIDRegex)

		if !valid.MatchString(v.(string)) {
			errors = append(errors, fmt.Errorf("Invalid %s specified (%q)", attrName, v.(string)))
		}

		return warnings, errors
	}
}

type urlParseFlags int

const (
	urlIsAbs urlParseFlags = 1 << iota
	urlOptional
	urlWithoutPath
	urlWithoutPort
	urlWithoutSchema
)

const urlBasicCheck urlParseFlags = 0

func validateHTTPURL(attrName schemaAttr, checkFlags urlParseFlags) func(v interface{}, key string) (warnings []string, errors []error) {
	return func(v interface{}, key string) (warnings []string, errors []error) {
		s := v.(string)
		if checkFlags&urlOptional != 0 && s == "" {
			return warnings, errors
		}

		u, err := url.Parse(v.(string))
		switch {
		case err != nil:
			errors = append(errors, errwrap.Wrapf(fmt.Sprintf("Invalid %s specified (%q): {{err}}", attrName, v.(string)), err))
		case u.Host == "":
			errors = append(errors, fmt.Errorf("Invalid %s specified: host can not be empty", attrName))
		case !(u.Scheme == "http" || u.Scheme == "https"):
			errors = append(errors, fmt.Errorf("Invalid %s specified: scheme unsupported (only support http and https)", attrName))
		}

		if checkFlags&urlIsAbs != 0 && !u.IsAbs() {
			errors = append(errors, fmt.Errorf("Schema is missing from URL %q (HINT: https://%s)", v.(string), v.(string)))
		}

		if checkFlags&urlWithoutSchema != 0 && u.IsAbs() {
			errors = append(errors, fmt.Errorf("Schema is present on URL %q (HINT: drop the https://%s)", v.(string), v.(string)))
		}

		if checkFlags&urlWithoutPath != 0 && u.Path != "" {
			errors = append(errors, fmt.Errorf("Path is present on URL %q (HINT: drop the %s)", v.(string), u.Path))
		}

		if checkFlags&urlWithoutPort != 0 {
			hostParts := strings.SplitN(u.Host, ":", 2)
			if len(hostParts) != 1 {
				errors = append(errors, fmt.Errorf("Port is present on URL %q (HINT: drop the :%s)", v.(string), hostParts[1]))
			}
		}

		return warnings, errors
	}
}

func validateStringIn(attrName schemaAttr, valid validStringValues) func(v interface{}, key string) (warnings []string, errors []error) {
	return func(v interface{}, key string) (warnings []string, errors []error) {
		s := v.(string)
		var found bool
		for i := range valid {
			if s == string(valid[i]) {
				found = true
				break
			}
		}

		if !found {
			errors = append(errors, fmt.Errorf("Invalid %q specified: %q not found in list %#v", string(attrName), s, valid))
		}

		return warnings, errors
	}
}
