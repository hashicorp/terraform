package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
)

// smcUserVariables does all the semantic checks to verify that the
// variables given satisfy the configuration itself.
func smcUserVariables(c *config.Config, vs map[string]string) []error {
	var errs []error

	cvs := make(map[string]*config.Variable)
	for _, v := range c.Variables {
		cvs[v.Name] = v
	}

	// Check that all required variables are present
	required := make(map[string]struct{})
	for _, v := range c.Variables {
		if v.Required() {
			required[v.Name] = struct{}{}
		}
	}
	for k, _ := range vs {
		delete(required, k)
	}
	if len(required) > 0 {
		for k, _ := range required {
			errs = append(errs, fmt.Errorf(
				"Required variable not set: %s", k))
		}
	}

	// Check that types match up
	for k, _ := range vs {
		v, ok := cvs[k]
		if !ok {
			continue
		}

		if v.Type() != config.VariableTypeString {
			errs = append(errs, fmt.Errorf(
				"%s: cannot assign string value to map type",
				k))
		}
	}

	// TODO(mitchellh): variables that are unknown

	return errs
}
