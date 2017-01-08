package circonus

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"text/scanner"
	"time"
	"unicode"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/errwrap"
)

var knownCheckTypes map[CheckType]struct{}
var knownContactMethods map[ContactMethods]struct{}

const (
	// Misc package constants
	defaultCheckTypeName  = "default"
	defaultNumHTTPHeaders = 3
)

var defaultCheckTypeConfigSize map[CheckType]int
var userContactMethods map[string]struct{}
var externalContactMethods map[string]struct{}

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

	userMethods := []string{"email", "sms", "xmpp"}
	externalMethods := []string{"slack"}

	knownContactMethods = make(map[ContactMethods]struct{}, len(externalContactMethods)+len(userContactMethods))

	externalContactMethods = make(map[string]struct{}, len(externalMethods))
	for _, k := range externalMethods {
		knownContactMethods[ContactMethods(k)] = struct{}{}
		externalContactMethods[k] = struct{}{}
	}

	userContactMethods = make(map[string]struct{}, len(userMethods))
	for _, k := range userMethods {
		knownContactMethods[ContactMethods(k)] = struct{}{}
		userContactMethods[k] = struct{}{}
	}
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

func validateContactMethod(v interface{}, key string) (warnings []string, errors []error) {
	if _, ok := knownContactMethods[ContactMethods(v.(string))]; !ok {
		warnings = append(warnings, fmt.Sprintf("Possibly unsupported contact method: %s", v.(string)))
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
		case cg.Escalations[severityIndex].After >= 0 && cg.Escalations[severityIndex].ContactGroupCID == "":
			return fmt.Errorf("severity %d escallation requires both and %s and %s be set", severityIndex+1, contactEscalateToAttr, contactEscalateAfterAttr)
		}
	}

	return nil
}

func validateContactGroupCID(attrName string) func(v interface{}, key string) (warnings []string, errors []error) {
	return func(v interface{}, key string) (warnings []string, errors []error) {
		validContactGroupCID := regexp.MustCompile(config.ContactGroupCIDRegex)

		if !validContactGroupCID.MatchString(v.(string)) {
			errors = append(errors, fmt.Errorf("Invalid %s specified (%q)", attrName, v.(string)))
		}

		return warnings, errors
	}
}

func validateDurationMin(attrName, minDuration string) func(v interface{}, key string) (warnings []string, errors []error) {
	var min time.Duration
	{
		var err error
		min, err = time.ParseDuration(minDuration)
		if err != nil {
			panic(fmt.Sprintf("Invalid time +%q: %v", minDuration, err))
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

func validateDurationMax(attrName, maxDuration string) func(v interface{}, key string) (warnings []string, errors []error) {
	var max time.Duration
	{
		var err error
		max, err = time.ParseDuration(maxDuration)
		if err != nil {
			panic(fmt.Sprintf("Invalid time +%q: %v", maxDuration, err))
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

func validateHTTPVersion(v interface{}, key string) (warnings []string, errors []error) {
	validHTTPVersion := regexp.MustCompile(`\d+\.\d+`)

	if !validHTTPVersion.MatchString(v.(string)) {
		errors = append(errors, fmt.Errorf("Invalid %s specified (%q)", checkConfigHTTPVersionAttr, v.(string)))
	}

	return warnings, errors
}

func validateIntMin(attrName string, min int) func(v interface{}, key string) (warnings []string, errors []error) {
	return func(v interface{}, key string) (warnings []string, errors []error) {
		if v.(int) < min {
			errors = append(errors, fmt.Errorf("Invalid %s specified (%d): minimum value must be %s", attrName, v.(int), min))
		}

		return warnings, errors
	}
}

func validateIntMax(attrName string, max int) func(v interface{}, key string) (warnings []string, errors []error) {
	return func(v interface{}, key string) (warnings []string, errors []error) {
		if v.(int) > max {
			errors = append(errors, fmt.Errorf("Invalid %s specified (%d): maximum value must be %s", attrName, v.(int), max))
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

func validateRegexp(attrName, reString string) func(v interface{}, key string) (warnings []string, errors []error) {
	re := regexp.MustCompile(reString)

	return func(v interface{}, key string) (warnings []string, errors []error) {
		if !re.MatchString(v.(string)) {
			errors = append(errors, fmt.Errorf("Invalid %s specified (%q): regexp failed", attrName, v.(string)))
		}

		return warnings, errors
	}
}

func validateTags(v interface{}, key string) (warnings []string, errors []error) {
	tagsRaw := v.(map[string]interface{})
	for k, valueRaw := range tagsRaw {
		{
			if len(k) == 0 {
				errors = append(errors, fmt.Errorf("tag category can not be empty"))
				continue
			}

			var s scanner.Scanner
			s.Init(strings.NewReader(k))
			var tok rune
		KEY:
			for tok != scanner.EOF {
				switch tok = s.Scan(); {
				case tok == ':':
					errors = append(errors, fmt.Errorf("tag category %q contains a colon character at codepoint %d", k, s.Pos()))
					break KEY
				case unicode.IsSpace(tok) == true:
					errors = append(errors, fmt.Errorf("tag category %+q contains a whitespace character at codepoint %d", k, s.Pos()))
					break KEY
				}
			}
		}

		{
			value := valueRaw.(string)
			if len(value) == 0 {
				continue
			}

			var s scanner.Scanner
			s.Init(strings.NewReader(value))
			var tok rune
		VALUE:
			for tok != scanner.EOF {
				switch tok = s.Scan(); {
				case tok == ':':
					errors = append(errors, fmt.Errorf("tag value %q contains a colon character at codepoint %d", value, s.Pos()))
					break VALUE
				case unicode.IsSpace(tok) == true:
					errors = append(errors, fmt.Errorf("tag value %q contains a whitespace character at codepoint %d", value, s.Pos()))
					break VALUE
				}
			}
		}
	}

	if numTags := len(tagsRaw); numTags > defaultWarnTags {
		warnings = append(warnings, fmt.Sprintf("Too many tags per resource (%d).  Recommend keeping it under %d", numTags, defaultWarnTags))
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

func validateHTTPURL(attrName string) func(v interface{}, key string) (warnings []string, errors []error) {

	return func(v interface{}, key string) (warnings []string, errors []error) {
		url, err := url.Parse(v.(string))
		switch {
		case err != nil:
			errors = append(errors, errwrap.Wrapf(fmt.Sprintf("Invalid %s specified (%q): {{err}}", attrName, v.(string)), err))
		case url.Host == "":
			errors = append(errors, fmt.Errorf("Invalid %s specified: host can not be empty", attrName))
		case !(url.Scheme == "http" || url.Scheme == "https"):
			errors = append(errors, fmt.Errorf("Invalid %s specified: scheme unsupported (only support http and https)", attrName))
		}

		return warnings, errors
	}
}

func validateStringIn(attrName _SchemaAttr, valid _ValidStringValues) func(v interface{}, key string) (warnings []string, errors []error) {
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
