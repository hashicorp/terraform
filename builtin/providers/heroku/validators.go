package heroku

import (
	"fmt"
	"strings"

	"github.com/satori/uuid"
)

func validatePipelineStageName(v interface{}, k string) (ws []string, errors []error) {
	validPipelineStageNames := []string{
		"review",
		"development",
		"staging",
		"production",
	}

	for _, s := range validPipelineStageNames {
		if v == s {
			return
		}
	}

	err := fmt.Errorf(
		"%s is an invalid pipeline stage, must be one of [%s]",
		v,
		strings.Join(validPipelineStageNames, ", "),
	)
	errors = append(errors, err)
	return
}

func validateUUID(v interface{}, k string) (ws []string, errors []error) {
	if _, err := uuid.FromString(v.(string)); err != nil {
		errors = append(errors, fmt.Errorf("%q is an invalid UUID: %s", k, err))
	}
	return
}
