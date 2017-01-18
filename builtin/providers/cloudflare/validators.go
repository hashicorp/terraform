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

func assertIsOneOf(setting string, acceptables []interface{}, value interface{}) error {
	for _, acceptable := range acceptables {
		if value == acceptable {
			return nil
		}
	}
	return fmt.Errorf("%q %q invalid: must be one of %q", setting, value, acceptables)
}

func validatePageRuleAction(v interface{}, k string) (ws []string, errors []error) {
	id := v.(map[string]interface{})["action"].(string)
	value := v.(map[string]interface{})["value"]

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
		errors = append(errors, fmt.Errorf("Value for %q action had type %q, expected %q", id, actualType, expectedType))
	}

	switch id {
	case "always_online":
	case "browser_check":
	case "email_obfuscation":
	case "ip_geolocation":
	case "server_side_exclude":
	case "smart_errors":
		if err := assertIsOneOf("Action status", []interface{}{"on", "off"}, value); err != nil {
			errors = append(errors, err)
		}
		break

	case "always_use_https":
	case "disable_apps":
	case "disable_performance":
	case "disable_security":
		if value != (struct{}{}) {
			ws = append(ws, fmt.Sprintf("Action %q does not take a value", id))
		}
		break

	case "browser_cache_ttl":
	case "edge_cache_ttl":
		maxTTL := 31536000
		if value.(int) > maxTTL {
			errors = append(errors, fmt.Errorf("Cache TTL too long: max value is %q", maxTTL))
		}
		break

	case "cache_level":
		if err := assertIsOneOf("Cache level", []interface{}{"bypass", "basic", "simplified", "aggressive", "cache_everything"}, value); err != nil {
			errors = append(errors, err)
		}
		break

	case "forwarding_url":
		forwardAction := value.(map[string]interface{})
		if reflect.TypeOf(forwardAction["url"]).Kind() != reflect.String {
			errors = append(errors, fmt.Errorf("Forwarding URL %q invalid: must be of type string", forwardAction["url"]))
		}
		if err := assertIsOneOf("Forwarding status code", []interface{}{301, 302}, forwardAction["status_code"]); err != nil {
			errors = append(errors, err)
		}
		break

	case "rocket_loader":
		if err := assertIsOneOf("Rocket loader", []interface{}{"off", "manual", "automatic"}, value); err != nil {
			errors = append(errors, err)
		}
		break

	case "security_level":
		if err := assertIsOneOf("Security level", []interface{}{"essentially_off", "low", "medium", "high", "under_attack"}, value); err != nil {
			errors = append(errors, err)
		}
		break

	case "ssl":
		if err := assertIsOneOf("SSL setting", []interface{}{"off", "flexible", "full", "strict"}, value); err != nil {
			errors = append(errors, err)
		}
		break

		/* The following action IDs are not yet implemented by cloudflare-go
		case "automatic_https_rewrites":
			return assertIsOnOrOff(value)
		case "opportunistic_encryption":
			return assertIsOnOrOff(value)*/
	}
	return
}
