package rundeck

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func validateValueFunc(values []string) schema.SchemaValidateFunc {
	return func(v interface{}, k string) (we []string, errors []error) {
		value := v.(string)
		valid := false
		for _, role := range values {
			if value == role {
				valid = true
				break
			}
		}

		if !valid {
			errors = append(errors, fmt.Errorf("%s is an invalid value for argument %s", value, k))
		}
		return
	}
}
