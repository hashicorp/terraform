package azurerm

import (
	"fmt"

	"github.com/satori/uuid"
	"regexp"
)

func validateName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[0-9A-Za-z_-]$`).MatchString(value) {
		errors = append(errors, fmt.Errorf("Only alphanumeric characters, `_` or `-` are allowed in %q: %q", k, value))
	}
	return
}

func validateUUID(v interface{}, k string) (ws []string, errors []error) {
	if _, err := uuid.FromString(v.(string)); err != nil {
		errors = append(errors, fmt.Errorf("%q is an invalid UUUID: %s", k, err))
	}
	return
}
