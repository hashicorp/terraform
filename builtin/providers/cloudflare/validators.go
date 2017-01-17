package cloudflare

import (
	"fmt"
	"net"
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
		"mirage":              {},
		"rocket_loader":       {},
		"security_level":      {},
		"server_side_exclude": {},
		"smart_errors":        {},
		"ssl":                 {},
		"waf":                 {},
	}

	if _, ok := validIDs[value]; !ok {
		errors = append(errors, fmt.Errorf(
			`%q contains an invalid action ID %q. Valid IDs are "always_online", "always_use_https", "browser_cache_ttl", "browser_check", "cache_level", "disable_apps", "disable_performance", "disable_railgun", "disable_security", "edge_cache_ttl", "email_obfuscation", "forwarding_url", "ip_geolocation", "mirage", "rocket_loader", "security_level", "server_side_exclude", "smart_errors", "ssl", or "waf"`, k, value))
	}
	return
}

func validatePageRuleActionValue(v interface{}, k string) (ws []string, errors []error) {
	return []string{}, []error{fmt.Errorf("Page Rule action value validation not implemented.")}
}
