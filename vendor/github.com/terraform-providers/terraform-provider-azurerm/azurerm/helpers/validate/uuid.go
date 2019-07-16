package validate

import (
	"fmt"
	"regexp"

	"github.com/hashicorp/go-uuid"
)

//uuid regex helper
var UUIDRegExp = regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$")

func UUID(i interface{}, k string) (warnings []string, errors []error) {
	v, ok := i.(string)
	if !ok {
		errors = append(errors, fmt.Errorf("expected type of %q to be string", k))
		return
	}

	if _, err := uuid.ParseUUID(v); err != nil {
		errors = append(errors, fmt.Errorf("%q isn't a valid UUID (%q): %+v", k, v, err))
	}

	return warnings, errors
}
func UUIDOrEmpty(i interface{}, k string) (warnings []string, errors []error) {
	v, ok := i.(string)
	if !ok {
		errors = append(errors, fmt.Errorf("expected type of %q to be string", k))
		return
	}

	if v == "" {
		return
	}

	return UUID(i, k)
}
