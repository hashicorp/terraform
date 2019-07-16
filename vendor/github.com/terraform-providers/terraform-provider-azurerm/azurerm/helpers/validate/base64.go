package validate

import (
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func Base64String() schema.SchemaValidateFunc {
	return func(i interface{}, k string) (warnings []string, errors []error) {
		// Empty string is not allowed
		if warnings, errors = NoEmptyStrings(i, k); len(errors) > 0 {
			return
		}

		v, ok := i.(string)
		if !ok {
			errors = append(errors, fmt.Errorf("expected type of %s to be string", k))
			return
		}

		if _, err := base64.StdEncoding.DecodeString(v); err != nil {
			errors = append(errors, fmt.Errorf("expect value (%s) of %s is base64 string", v, k))
		}

		return
	}
}
