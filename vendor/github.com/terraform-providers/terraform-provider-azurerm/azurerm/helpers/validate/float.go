package validate

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

// FloatAtLeast returns a SchemaValidateFunc which tests if the provided value
// is of type float64 and is at least min (inclusive)
func FloatAtLeast(min float64) schema.SchemaValidateFunc {
	return func(i interface{}, k string) (_ []string, errors []error) {
		v, ok := i.(float64)
		if !ok {
			errors = append(errors, fmt.Errorf("expected type of %s to be float64", k))
			return nil, errors
		}

		if v < min {
			errors = append(errors, fmt.Errorf("expected %s to be at least (%f), got %f", k, min, v))
			return nil, errors
		}

		return nil, errors
	}
}
