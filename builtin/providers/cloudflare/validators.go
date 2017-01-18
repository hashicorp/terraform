package cloudflare

import (
	"fmt"
	"net"
	"reflect"
	"strings"
)

// validateRecordType ensures that the cloudflare record type is valid
func validateRecordType(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	validTypes := map[string]struct{}{
		"A":     {},
		"AAAA":  {},
		"CNAME": {},
		"TXT":   {},
		"SRV":   {},
		"LOC":   {},
		"MX":    {},
		"NS":    {},
		"SPF":   {},
	}

	if _, ok := validTypes[value]; !ok {
		errors = append(errors, fmt.Errorf(
			`%q contains an invalid type %q. Valid types are "A", "AAAA", "CNAME", "TXT", "SRV", "LOC", "MX", "NS" or "SPF"`, k, value))
	}
	return
}

// validateRecordName ensures that based on supplied record type, the name content matches
// Currently only validates A and AAAA types
func validateRecordName(t string, value string) error {
	switch t {
	case "A":
		// Must be ipv4 addr
		addr := net.ParseIP(value)
		if addr == nil || !strings.Contains(value, ".") {
			return fmt.Errorf("A record must be a valid IPv4 address, got: %q", value)
		}
	case "AAAA":
		// Must be ipv6 addr
		addr := net.ParseIP(value)
		if addr == nil || !strings.Contains(value, ":") {
			return fmt.Errorf("AAAA record must be a valid IPv6 address, got: %q", value)
		}
	}

	return nil
}

func validatePageRuleStatus(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	validStatuses := map[string]struct{}{
		"active": {},
		"paused": {},
	}

	if _, ok := validStatuses[value]; !ok {
		errors = append(errors, fmt.Errorf(
			`%q contains an invalid status %q. Valid statuses are "active" or "paused"`, k, value))
	}
	return
}

func validatePageRuleActionID(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	validIDs := map[string]struct{}{
		"always_online":       {},
		"always_use_https":    {},
		"browser_cache_ttl":   {},
		"browser_check":       {},
		"cache_level":         {},
		"disable_apps":        {},
		"disable_performance": {},
		"disable_railgun":     {},
		"disable_security":    {},
		"edge_cache_ttl":      {},
		"email_obfuscation":   {},
		"forwarding_url":      {},
		"ip_geolocation":      {},
		"rocket_loader":       {},
		"security_level":      {},
		"server_side_exclude": {},
		"smart_errors":        {},
		"ssl":                 {},
		/* The following action IDs are not yet implemented by cloudflare-go
		   "automatic_https_rewrites": reflect.String,
		   "opportunistic_encryption": reflect.String,*/
	}

	if _, ok := validIDs[value]; !ok {
		errors = append(errors, fmt.Errorf(
			`%q contains an invalid action ID %q. Valid IDs are "always_online", "always_use_https", "browser_cache_ttl", "browser_check", "cache_level", "disable_apps", "disable_performance", "disable_railgun", "disable_security", "edge_cache_ttl", "email_obfuscation", "forwarding_url", "ip_geolocation", "mirage", "rocket_loader", "security_level", "server_side_exclude", "smart_errors", "ssl", or "waf"`, k, value))
	}
	return
}

func assertIsOnOrOff(value interface{}) error {
	return assertIsOneOf("Action status", []interface{}{"on", "off"}, value)
}

func assertIsOneOf(setting string, acceptables []interface{}, value interface{}) error {
	for _, acceptable := range acceptables {
		if value == acceptable {
			return nil
		}
	}
	return fmt.Errorf("%q %q invalid: must be one of %q", setting, value, acceptables)
}

func assertIsUnitary(id string, value interface{}) error {
	if value != (struct{}{}) {
		return fmt.Errorf("Action %q does not take a value", id)
	}
	return nil
}

func validatePageRuleActionValue(id string, value interface{}) error {
	expectedTypeFor := map[string]reflect.Kind{
		"always_online":       reflect.String,
		"always_use_https":    reflect.Interface,
		"browser_cache_ttl":   reflect.Int,
		"browser_check":       reflect.String,
		"cache_level":         reflect.String,
		"disable_apps":        reflect.Interface,
		"disable_performance": reflect.Interface,
		"disable_railgun":     reflect.String,
		"disable_security":    reflect.Interface,
		"edge_cache_ttl":      reflect.Int,
		"email_obfuscation":   reflect.String,
		"forwarding_url":      reflect.Map,
		"ip_geolocation":      reflect.String,
		"rocket_loader":       reflect.String,
		"security_level":      reflect.String,
		"server_side_exclude": reflect.String,
		"smart_errors":        reflect.String,
		"ssl":                 reflect.String,
		/* The following action IDs are not yet implemented by cloudflare-go
		   "automatic_https_rewrites": reflect.String,
		   "opportunistic_encryption": reflect.String,*/
	}

	actualType := reflect.TypeOf(value).Kind()
	expectedType := expectedTypeFor[id]
	if actualType != expectedType {
		return fmt.Errorf("Value for %q action had type %q, expected %q", id, actualType, expectedType)
	}

	switch id {
	default:
		return nil
	case "always_online":
		return assertIsOnOrOff(value)
	case "always_use_https":
		return assertIsUnitary(id, value)
	case "browser_check":
		return assertIsOnOrOff(value)
	case "cache_level":
		return assertIsOneOf("Cache level", []interface{}{"bypass", "basic", "simplified", "aggressive", "cache_everything"}, value)
	case "disable_apps":
		return assertIsUnitary(id, value)
	case "disable_performance":
		return assertIsUnitary(id, value)
	case "disable_security":
		return assertIsUnitary(id, value)
	case "email_obfuscation":
		return assertIsOnOrOff(value)
	case "forwarding_url":
		forwardAction := value.(map[string]interface{})
		if reflect.TypeOf(forwardAction["url"]).Kind() != reflect.String {
			return fmt.Errorf("Forwarding URL %q invalid: must be of type string", forwardAction["url"])
		}
		return assertIsOneOf("Forwarding status code", []interface{}{301, 302}, forwardAction["status_code"])
	case "ip_geolocation":
		return assertIsOnOrOff(value)
	case "rocket_loader":
		return assertIsOneOf("Rocket loader", []interface{}{"off", "manual", "automatic"}, value)
	case "security_level":
		return assertIsOneOf("Security level", []interface{}{"essentially_off", "low", "medium", "high", "under_attack"}, value)
	case "server_side_exclude":
		return assertIsOnOrOff(value)
	case "smart_errors":
		return assertIsOnOrOff(value)
	case "ssl":
		return assertIsOneOf("SSL setting", []interface{}{"off", "flexible", "full", "strict"}, value)
		/* The following action IDs are not yet implemented by cloudflare-go
		   case "automatic_https_rewrites":
		       return assertIsOnOrOff(value)
		   case "opportunistic_encryption":
		       return assertIsOnOrOff(value)*/
	}
}
