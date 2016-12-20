package arukas

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"strings"
)

func validateMaxLength(minLength, maxLength int) schema.SchemaValidateFunc {
	return func(v interface{}, k string) (ws []string, errors []error) {
		value := v.(string)
		if len(value) < minLength {
			errors = append(errors, fmt.Errorf(
				"%q cannot be shorter than %d characters: %q", k, minLength, value))
		}
		if len(value) > maxLength {
			errors = append(errors, fmt.Errorf(
				"%q cannot be longer than %d characters: %q", k, maxLength, value))
		}
		return
	}
}

func validateIntegerInRange(min, max int) schema.SchemaValidateFunc {
	return func(v interface{}, k string) (ws []string, errors []error) {
		value := v.(int)
		if value < min {
			errors = append(errors, fmt.Errorf(
				"%q cannot be lower than %d: %d", k, min, value))
		}
		if value > max {
			errors = append(errors, fmt.Errorf(
				"%q cannot be higher than %d: %d", k, max, value))
		}
		return
	}
}

func validateStringInWord(allowWords []string) schema.SchemaValidateFunc {
	return func(v interface{}, k string) (ws []string, errors []error) {
		var found bool
		for _, t := range allowWords {
			if v.(string) == t {
				found = true
			}
		}
		if !found {
			errors = append(errors, fmt.Errorf("%q must be one of [%s]", k, strings.Join(allowWords, "/")))

		}
		return
	}
}

func validateIntInWord(allowWords []string) schema.SchemaValidateFunc {
	return func(v interface{}, k string) (ws []string, errors []error) {
		var found bool
		for _, t := range allowWords {
			if fmt.Sprintf("%d", v.(int)) == t {
				found = true
			}
		}
		if !found {
			errors = append(errors, fmt.Errorf("%q must be one of [%s]", k, strings.Join(allowWords, "/")))

		}
		return
	}
}

func validateDNSRecordValue() schema.SchemaValidateFunc {
	return func(v interface{}, k string) (ws []string, errors []error) {
		var rtype, value string

		values := v.(map[string]interface{})
		rtype = values["type"].(string)
		value = values["value"].(string)
		switch rtype {
		case "MX", "NS", "CNAME":
			if rtype == "MX" {
				if values["priority"] == nil {
					errors = append(errors, fmt.Errorf("%q required when TYPE was MX", k))
				}
			}
			if !strings.HasSuffix(value, ".") {
				errors = append(errors, fmt.Errorf("%q must be period at the end [%s]", k, value))
			}
		}
		return
	}

}
