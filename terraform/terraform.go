package terraform

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/config"
)

// Terraform is the primary structure that is used to interact with
// Terraform from code, and can perform operations such as returning
// all resources, a resource tree, a specific resource, etc.
type Terraform struct {
	config  *config.Config
	mapping map[*config.Resource]ResourceProvider
}

// Config is the configuration that must be given to instantiate
// a Terraform structure.
type Config struct {
	Config    *config.Config
	Providers map[string]ResourceProviderFactory
	Variables map[string]string
}

// New creates a new Terraform structure, initializes resource providers
// for the given configuration, etc.
//
// Semantic checks of the entire configuration structure are done at this
// time, as well as richer checks such as verifying that the resource providers
// can be properly initialized, can be configured, etc.
func New(c *Config) (*Terraform, error) {
	// Go through each resource and match it up to a provider
	mapping := make(map[*config.Resource]ResourceProvider)
	providers := make(map[string]ResourceProvider)
	for _, r := range c.Config.Resources {
		// Find the prefixes that match this in the order of
		// longest matching first (most specific)
		prefixes := matchingPrefixes(r.Type, c.Providers)

		// Go through each prefix and instantiate if necessary, then
		// verify if this provider is of use to us or not.
		var provider ResourceProvider = nil
		for _, prefix := range prefixes {
			p, ok := providers[prefix]
			if !ok {
				var err error
				p, err = c.Providers[prefix]()
				if err != nil {
					err = fmt.Errorf(
						"Error instantiating resource provider for "+
							"prefix %s: %s", prefix, err)
					return nil, err
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
			return nil, fmt.Errorf(
				"Provider for resource %s not found.",
				r.Id())
		}

		mapping[r] = provider
	}

	return &Terraform{
		config:  c.Config,
		mapping: mapping,
	}, nil
}

func (t *Terraform) Apply(*State, *Diff) (*State, error) {
	return nil, nil
}

func (t *Terraform) Diff(*State) (*Diff, error) {
	return nil, nil
}

func (t *Terraform) Refresh(*State) (*State, error) {
	return nil, nil
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
