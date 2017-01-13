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
