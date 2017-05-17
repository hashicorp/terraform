package pagerduty

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

// Validate a value against a set of possible values
func validateValueFunc(values []string) schema.SchemaValidateFunc {
	return func(v interface{}, k string) (we []string, errors []error) {
		value := v.(string)
		valid := false
		for _, val := range values {
			if value == val {
				valid = true
				break
			}
		}

		if !valid {
			errors = append(errors, fmt.Errorf("%#v is an invalid value for argument %s. Must be one of %#v", value, k, values))
		}
		return
	}
}
