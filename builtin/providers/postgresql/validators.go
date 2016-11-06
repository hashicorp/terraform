package postgresql

import "fmt"

func validateConnLimit(v interface{}, key string) (warnings []string, errors []error) {
	value := v.(int)
	if value < -1 {
		errors = append(errors, fmt.Errorf("%d can not be less than -1", key))
	}
	return
}
