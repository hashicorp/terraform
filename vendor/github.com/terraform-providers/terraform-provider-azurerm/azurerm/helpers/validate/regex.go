package validate

import (
	"fmt"
	"regexp"
)

func RegExHelper(i interface{}, k, r string) (match bool, errors []error) {
	v, ok := i.(string)
	if !ok {
		return false, []error{fmt.Errorf("expected type of %q to be string", k)}
	}

	if regexp.MustCompile(r).MatchString(v) {
		return true, nil
	}

	return false, []error{fmt.Errorf("%q did not match regex %q", k, r)}
}
