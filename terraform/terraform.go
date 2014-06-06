package terraform

import (
	"fmt"
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
	mapping   map[*config.Resource]*terraformProvider
	variables map[string]string
}

// terraformProvider contains internal state information about a resource
// provider for Terraform.
type terraformProvider struct {
	Provider ResourceProvider
	Config   *config.ProviderConfig

	sync.Once
}

// Config is the configuration that must be given to instantiate
// a Terraform structure.
type Config struct {
	Config    *config.Config
	Providers map[string]ResourceProviderFactory
	Variables map[string]string

	computedPlaceholder string
}

// New creates a new Terraform structure, initializes resource providers
// for the given configuration, etc.
//
// Semantic checks of the entire configuration structure are done at this
// time, as well as richer checks such as verifying that the resource providers
// can be properly initialized, can be configured, etc.
func New(c *Config) (*Terraform, error) {
	var errs []error

	// Calculate the computed key placeholder
	c.computedPlaceholder = "tf_computed_placeholder"

	// Validate that all required variables have values
	if err := smcVariables(c); err != nil {
		errs = append(errs, err...)
	}

	// Match all the resources with a provider and initialize the providers
	mapping, err := smcProviders(c)
	if err != nil {
		errs = append(errs, err...)
	}

	// Build the resource graph
	graph := c.Config.Graph()
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
	var l sync.RWMutex

	// Initialize the result diff so we can write to it
	result.init()

	// Initialize the variables for application
	vars := make(map[string]string)
	for k, v := range t.variables {
		vars[k] = v
	}

	return func(n *depgraph.Noun) error {
		// If it is the root node, ignore
		if n.Meta == nil {
			return nil
		}

		switch n.Meta.(type) {
		case *config.ProviderConfig:
			// Ignore, we don't treat this any differently since we always
			// initialize the provider on first use and use a lock to make
			// sure we only do this once.
			return nil
		case *config.Resource:
			// Continue
		}

		r := n.Meta.(*config.Resource)
		p := t.mapping[r]
		if p == nil {
			panic(fmt.Sprintf("No provider for resource: %s", r.Id()))
		}

		// Initialize the provider if we haven't already
		p.init(vars)

		l.RLock()
		var rs *ResourceState
		if state != nil {
			rs = state.resources[r.Id()]
		}
		if len(vars) > 0 {
			r = r.ReplaceVariables(vars)
		}
		l.RUnlock()

		diff, err := p.Provider.Diff(rs, r.Config)
		if err != nil {
			return err
		}

		// If there were no diff items, return right away
		if len(diff.Attributes) == 0 {
			return nil
		}

		// Acquire a lock since this function is called in parallel
		l.Lock()
		defer l.Unlock()

		// Update the resulting diff
		result.Resources[r.Id()] = diff.Attributes

		// Determine the new state and update variables
		rs = rs.MergeDiff(diff.Attributes, ComputedPlaceholder)
		for ak, av := range rs.Attributes {
			vars[fmt.Sprintf("%s.%s", r.Id(), ak)] = av
		}

		return nil
	}
}

func (t *terraformProvider) init(vars map[string]string) error {
	var err error

	t.Once.Do(func() {
		var c map[string]interface{}
		if t.Config != nil {
			c = t.Config.ReplaceVariables(vars).Config
		}

		_, err = t.Provider.Configure(c)
	})

	return err
}
