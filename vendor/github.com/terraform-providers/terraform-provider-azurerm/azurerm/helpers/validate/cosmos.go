package validate

import (
	"fmt"
	"regexp"
)

func CosmosAccountName(v interface{}, k string) (warnings []string, errors []error) {
	value := v.(string)

	// Portal: The value must contain only alphanumeric characters or the following: -
	if matched := regexp.MustCompile("^[-a-z0-9]{3,50}$").Match([]byte(value)); !matched {
		errors = append(errors, fmt.Errorf("%s name must be 3 - 50 characters long, contain only letters, numbers and hyphens.", k))
	}

	return warnings, errors
}

func CosmosEntityName(v interface{}, k string) (warnings []string, errors []error) {
	value := v.(string)

	if len(value) < 1 || len(value) > 255 {
		errors = append(errors, fmt.Errorf(
			"%q must be between 1 and 255 characters: %q", k, value))
	}

	return warnings, errors
}
