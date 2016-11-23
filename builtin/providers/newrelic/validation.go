package newrelic

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func float64Gte(gte float64) schema.SchemaValidateFunc {
	return func(i interface{}, k string) (s []string, es []error) {
		v, ok := i.(float64)
		if !ok {
			es = append(es, fmt.Errorf("expected type of %s to be float64", k))
			return
		}

		if v >= gte {
			return
		}

		es = append(es, fmt.Errorf("expected %s to be greater than or equal to %v, got %v", k, gte, v))
		return
	}
}

func intInSlice(valid []int) schema.SchemaValidateFunc {
	return func(i interface{}, k string) (s []string, es []error) {
		v, ok := i.(int)
		if !ok {
			es = append(es, fmt.Errorf("expected type of %s to be int", k))
			return
		}

		for _, p := range valid {
			if v == p {
				return
			}
		}

		es = append(es, fmt.Errorf("expected %s to be one of %v, got %v", k, valid, v))
		return
	}
}
