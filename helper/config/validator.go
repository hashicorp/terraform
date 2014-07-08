package config

import (
	"fmt"

	"github.com/hashicorp/terraform/terraform"
)

// Validator is a helper that helps you validate the configuration
// of your resource, resource provider, etc.
type Validator struct {
	Required []string
	Optional []string
}

func (v *Validator) Validate(
	c *terraform.ResourceConfig) (ws []string, es []error) {
	keySet := make(map[string]bool)
	reqSet := make(map[string]struct{})
	for _, k := range v.Required {
		keySet[k] = true
		reqSet[k] = struct{}{}
	}
	for _, k := range v.Optional {
		keySet[k] = false
	}

	// Find any unknown keys used and mark off required keys that
	// we have set.
	for k, _ := range c.Raw {
		_, ok := keySet[k]
		if !ok {
			es = append(es, fmt.Errorf(
				"Unknown configuration key: %s", k))
			continue
		}

		delete(reqSet, k)
	}

	// Check what keys are required that we didn't set
	for k, _ := range reqSet {
		es = append(es, fmt.Errorf(
			"Required key is not set: %s", k))
	}

	return
}
