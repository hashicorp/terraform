package fastly

import "fmt"

func validateLoggingFormatVersion(v interface{}, k string) (ws []string, errors []error) {
	value := uint(v.(int))
	validVersions := map[uint]struct{}{
		1: {},
		2: {},
	}

	if _, ok := validVersions[value]; !ok {
		errors = append(errors, fmt.Errorf(
			"%q must be one of ['1', '2']", k))
	}
	return
}

func validateLoggingMessageType(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	validTypes := map[string]struct{}{
		"classic": {},
		"loggly":  {},
		"logplex": {},
		"blank":   {},
	}

	if _, ok := validTypes[value]; !ok {
		errors = append(errors, fmt.Errorf(
			"%q must be one of ['classic', 'loggly', 'logplex', 'blank']", k))
	}
	return
}
