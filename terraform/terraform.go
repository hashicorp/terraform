package terraform

import (
	"fmt"
	"strings"
	"sync"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/depgraph"
)

// Terraform is the primary structure that is used to interact with
// Terraform from code, and can perform operations such as returning
// all resources, a resource tree, a specific resource, etc.
type Terraform struct {
	config    *config.Config
	graph     *depgraph.Graph
	mapping   map[*config.Resource]ResourceProvider
	variables map[string]string
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
	var errs []error

	// Validate that all required variables have values
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
			errs = append(errs, fmt.Errorf(
				"Provider for resource %s not found.",
				r.Id()))
		}

		mapping[r] = provider
	}

	// Build the resource graph
	graph := c.Config.ResourceGraph()
	if err := graph.Validate(); err != nil {
		errs = append(errs, fmt.Errorf(
			"Resource graph has an error: %s", err))
	}

	// If we accumulated any errors, then return them all
	if len(errs) > 0 {
		return nil, &MultiError{Errors: errs}
	}

	return &Terraform{
		config:    c.Config,
		graph:     graph,
		mapping:   mapping,
		variables: c.Variables,
	}, nil
}

func (t *Terraform) Apply(*State, *Diff) (*State, error) {
	return nil, nil
}

func (t *Terraform) Diff(s *State) (*Diff, error) {
	result := new(Diff)
	err := t.graph.Walk(t.diffWalkFn(s, result))
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (t *Terraform) Refresh(*State) (*State, error) {
	return nil, nil
}

func (t *Terraform) diffWalkFn(
	state *State, result *Diff) depgraph.WalkFunc {
	var resultLock sync.Mutex

	return func(n *depgraph.Noun) error {
		// If it is the root node, ignore
		if n.Name == config.ResourceGraphRoot {
			return nil
		}

		r := n.Meta.(*config.Resource)
		p := t.mapping[r]
		if p == nil {
			panic(fmt.Sprintf("No provider for resource: %s", r.Id()))
		}

		var rs ResourceState
		diff, err := p.Diff(rs, r.Config)
		if err != nil {
			return err
		}

		// If there were no diff items, return right away
		if len(diff.Attributes) == 0 {
			return nil
		}

		// Acquire a lock and modify the resulting diff
		resultLock.Lock()
		defer resultLock.Unlock()
		result.init()
		result.Resources[r.Id()] = diff.Attributes

		return nil
	}
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
