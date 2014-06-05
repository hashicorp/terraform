package terraform

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/config"
)

// smcProviders matches up the resources with a provider and initializes
// it. This does not call "Configure" on the ResourceProvider, since that
// might actually depend on upstream resources.
func smcProviders(c *Config) (map[*config.Resource]ResourceProvider, []error) {
	var errs []error

	// Keep track of providers we know we couldn't instantiate so
	// that we don't get a ton of errors about the same provider.
	failures := make(map[string]struct{})

	// Go through each resource and match it up to a provider
	mapping := make(map[*config.Resource]ResourceProvider)
	providers := make(map[string]ResourceProvider)

ResourceLoop:
	for _, r := range c.Config.Resources {
		// Find the prefixes that match this in the order of
		// longest matching first (most specific)
		prefixes := matchingPrefixes(r.Type, c.Providers)

		if len(prefixes) > 0 {
			if _, ok := failures[prefixes[0]]; ok {
				// We already failed this provider, meaning this
				// resource will never succeed, so just continue.
				continue
			}
		}

		// Go through each prefix and instantiate if necessary, then
		// verify if this provider is of use to us or not.
		var provider ResourceProvider = nil
		for _, prefix := range prefixes {
			// Initialize the provider
			p, ok := providers[prefix]
			if !ok {
				var err error
				p, err = c.Providers[prefix]()
				if err != nil {
					errs = append(errs, fmt.Errorf(
						"Error instantiating resource provider for "+
							"prefix %s: %s", prefix, err))

					// Record the error so that we don't check it again
					failures[prefix] = struct{}{}

					// Jump to the next resource
					continue ResourceLoop
				}

				providers[prefix] = p
			}

			// Test if this provider matches what we need
			if !ProviderSatisfies(p, r.Type) {
				continue
			}

			// A match! Set it and break
			provider = p
			break
		}

		if provider == nil {
			// We never found a matching provider.
			errs = append(errs, fmt.Errorf(
				"Provider for resource %s not found.",
				r.Id()))
		}

		mapping[r] = provider
	}

	if len(errs) > 0 {
		return nil, errs
	}

	return mapping, nil
}

// smcVariables does all the semantic checks to verify that the
// variables given in the configuration to instantiate a Terraform
// struct are valid.
func smcVariables(c *Config) []error {
	var errs []error

	// Check that all required variables are present
	required := make(map[string]struct{})
	for k, v := range c.Config.Variables {
		if v.Required() {
			required[k] = struct{}{}
		}
	}
	for k, _ := range c.Variables {
		delete(required, k)
	}
	if len(required) > 0 {
		for k, _ := range required {
			errs = append(errs, fmt.Errorf(
				"Required variable not set: %s", k))
		}
	}

	// TODO(mitchellh): variables that are unknown

	return errs
}

// matchingPrefixes takes a resource type and a set of resource
// providers we know about by prefix and returns a list of prefixes
// that might be valid for that resource.
//
// The list returned is in the order that they should be attempted.
func matchingPrefixes(
	t string,
	ps map[string]ResourceProviderFactory) []string {
	result := make([]string, 0, 1)
	for prefix, _ := range ps {
		if strings.HasPrefix(t, prefix) {
			result = append(result, prefix)
		}
	}

	// TODO(mitchellh): Order by longest prefix first

	return result
}
