package fastly

import "fmt"

func validateS3FormatVersion(v interface{}, k string) (ws []string, errors []error) {
	value := uint(v.(int))
	validVersions := map[uint]struct{}{
		1: {},
		2: {},
	}

	if _, ok := validVersions[value]; !ok {
		errors = append(errors, fmt.Errorf(
			"%q must be one of ['1', '2']", k))
	}
	return
}
