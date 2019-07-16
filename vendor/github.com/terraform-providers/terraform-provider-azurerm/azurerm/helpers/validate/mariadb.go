package validate

import (
	"fmt"
	"regexp"
)

func MariaDBFirewallRuleName(v interface{}, k string) (warnings []string, errors []error) {
	value := v.(string)

	// Firewall rule name can contain alphanumeric characters and hyphens and must be 1 - 128 characters long
	if matched := regexp.MustCompile("^[-a-z0-9]{1,128}$").Match([]byte(value)); !matched {
		errors = append(errors, fmt.Errorf("Firewall rule name must be 1 - 128 characters long, contain only letters, numbers and hyphens."))
	}

	return warnings, errors
}

func MariaDBServerName(v interface{}, k string) (warnings []string, errors []error) {
	value := v.(string)

	// MariaDB server name can contain alphanumeric characters and hyphens and must be 3 - 63 characters long
	if matched := regexp.MustCompile("^[-a-z0-9]{3,63}$").Match([]byte(value)); !matched {
		errors = append(errors, fmt.Errorf("Server name must be 3 - 63 characters long, contain only letters, numbers and hyphens."))
	}

	return warnings, errors
}
