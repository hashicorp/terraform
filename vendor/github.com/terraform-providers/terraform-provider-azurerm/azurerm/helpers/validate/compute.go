package validate

import (
	"fmt"
	"regexp"
)

func SharedImageGalleryName(v interface{}, k string) (warnings []string, errors []error) {
	value := v.(string)
	// Image gallery name accepts only alphanumeric, dots and underscores in the name (no dashes)
	r, _ := regexp.Compile(`^[A-Za-z0-9._]+$`)
	if !r.MatchString(value) {
		errors = append(errors, fmt.Errorf("%s can only contain alphanumeric, full stops and underscores. Got %q.", k, value))
	}

	length := len(value)
	if length >= 80 {
		errors = append(errors, fmt.Errorf("%s can be up to 80 characters, currently %d.", k, length))
	}

	return warnings, errors
}

func SharedImageName(v interface{}, k string) (warnings []string, errors []error) {
	// different from the shared image gallery name
	value := v.(string)

	r, _ := regexp.Compile(`^[A-Za-z0-9._-]+$`)
	if !r.MatchString(value) {
		errors = append(errors, fmt.Errorf("%s can only contain alphanumeric, full stops, dashes and underscores. Got %q.", k, value))
	}

	length := len(value)
	if length >= 80 {
		errors = append(errors, fmt.Errorf("%s can be up to 80 characters, currently %d.", k, length))
	}

	return warnings, errors
}

func SharedImageVersionName(v interface{}, k string) (warnings []string, errors []error) {
	value := v.(string)

	r, _ := regexp.Compile(`^([0-9]{1,10}\.[0-9]{1,10}\.[0-9]{1,10})$`)
	if !r.MatchString(value) {
		errors = append(errors, fmt.Errorf("Expected %s to be in the format `1.2.3` but got %q.", k, value))
	}

	return warnings, errors
}
