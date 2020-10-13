package validation

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

// IntBetween returns a SchemaValidateFunc which tests if the provided value
// is of type int and is between min and max (inclusive)
func IntBetween(min, max int) schema.SchemaValidateFunc {
	return func(i interface{}, k string) (s []string, es []error) {
		v, ok := i.(int)
		if !ok {
			es = append(es, fmt.Errorf("expected type of %s to be int", k))
			return
		}

		if v < min || v > max {
			es = append(es, fmt.Errorf("expected %s to be in the range (%d - %d), got %d", k, min, max, v))
			return
		}

		return
	}
}

// StringInSlice returns a SchemaValidateFunc which tests if the provided value
// is of type string and matches the value of an element in the valid slice
// will test with in lower case if ignoreCase is true
func StringInSlice(valid []string, ignoreCase bool) schema.SchemaValidateFunc {
	return func(i interface{}, k string) (s []string, es []error) {
		v, ok := i.(string)
		if !ok {
			es = append(es, fmt.Errorf("expected type of %s to be string", k))
			return
		}

		for _, str := range valid {
			if v == str || (ignoreCase && strings.ToLower(v) == strings.ToLower(str)) {
				return
			}
		}

		es = append(es, fmt.Errorf("expected %s to be one of %v, got %s", k, valid, v))
		return
	}
}
