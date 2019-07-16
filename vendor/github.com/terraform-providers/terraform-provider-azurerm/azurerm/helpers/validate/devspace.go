package validate

import (
	"regexp"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func DevSpaceName() schema.SchemaValidateFunc {
	return func(i interface{}, k string) (warnings []string, errors []error) {
		// Length should be between 3 and 31.
		if warnings, errors = validation.StringLenBetween(3, 31)(i, k); len(errors) > 0 {
			return warnings, errors
		}

		// Naming rule.
		regexStr := "^[a-zA-Z0-9](-?[a-zA-Z0-9])*$"
		errMsg := "DevSpace name can only include alphanumeric characters, hyphens."
		if warnings, errors = validation.StringMatch(regexp.MustCompile(regexStr), errMsg)(i, k); len(errors) > 0 {
			return warnings, errors
		}

		return warnings, errors
	}
}
