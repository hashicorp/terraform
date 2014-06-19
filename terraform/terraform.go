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

// This is a function type used to implement a walker for the resource
// tree internally on the Terraform structure.
type genericWalkFunc func(*Resource) (map[string]string, error)

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

	// Validate all the configurations, once.
	tps := make(map[*terraformProvider]struct{})
	for _, tp := range mapping {
		if _, ok := tps[tp]; !ok {
			tps[tp] = struct{}{}
		}
	}
	for tp, _ := range tps {
		var rc *ResourceConfig
		if tp.Config != nil {
			rc = NewResourceConfig(tp.Config.RawConfig)
		}

		_, tpErrs := tp.Provider.Validate(rc)
		if len(tpErrs) > 0 {
			errs = append(errs, tpErrs...)
		}
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

func (t *Terraform) Apply(s *State, d *Diff) (*State, error) {
	result := new(State)
	err := t.graph.Walk(t.applyWalkFn(s, d, result))
	if err != nil {
		return nil, err
	}

	return result, nil
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

func (t *Terraform) applyWalkFn(
	state *State,
	diff *Diff,
	result *State) depgraph.WalkFunc {
	var l sync.Mutex

	// Initialize the result
	result.init()

	cb := func(r *Resource) (map[string]string, error) {
		rs, err := r.Provider.Apply(r.State, r.Diff)
		if err != nil {
			return nil, err
		}

		// If no state was returned, then no variables were updated so
		// just return.
		if rs == nil {
			return nil, nil
		}

		// Update the resulting diff
		l.Lock()
		result.Resources[r.Id] = rs
		l.Unlock()

		// Determine the new state and update variables
		vars := make(map[string]string)
		for ak, av := range rs.Attributes {
			vars[fmt.Sprintf("%s.%s", r.Id, ak)] = av
		}

		return vars, nil
	}

	return t.genericWalkFn(state, diff, cb)
}

func (t *Terraform) diffWalkFn(
	state *State, result *Diff) depgraph.WalkFunc {
	var l sync.Mutex

	// Initialize the result diff so we can write to it
	result.init()

	cb := func(r *Resource) (map[string]string, error) {
		diff, err := r.Provider.Diff(r.State, r.Config)
		if err != nil {
			return nil, err
		}

		// If there were no diff items, return right away
		if diff == nil || len(diff.Attributes) == 0 {
			return nil, nil
		}

		// Update the resulting diff
		l.Lock()
		result.Resources[r.Id] = diff
		l.Unlock()

		// Determine the new state and update variables
		vars := make(map[string]string)
		rs := r.State.MergeDiff(diff.Attributes)
		for ak, av := range rs.Attributes {
			vars[fmt.Sprintf("%s.%s", r.Id, ak)] = av
		}

		return vars, nil
	}

	return t.genericWalkFn(state, nil, cb)
}

func (t *Terraform) genericWalkFn(
	state *State,
	diff *Diff,
	cb genericWalkFunc) depgraph.WalkFunc {
	var l sync.Mutex

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
		if err := p.init(vars); err != nil {
			return err
		}

		// Get the resource state
		var rs *ResourceState
		if state != nil {
			rs = state.Resources[r.Id()]
		}

		// Get the resource diff
		var rd *ResourceDiff
		if diff != nil {
			rd = diff.Resources[r.Id()]
		}

		if len(vars) > 0 {
			if err := r.RawConfig.Interpolate(vars); err != nil {
				panic(fmt.Sprintf("Interpolate error: %s", err))
			}
		}

		// If we have no state, then create an empty state with the
		// type fulfilled at the least.
		if rs == nil {
			rs = new(ResourceState)
		}
		rs.Type = r.Type

		// Call the callack
		newVars, err := cb(&Resource{
			Id: r.Id(),
			Config: &ResourceConfig{
				ComputedKeys: r.RawConfig.UnknownKeys(),
				Raw:          r.RawConfig.Config(),
			},
			Diff:     rd,
			Provider: p.Provider,
			State:    rs,
		})
		if err != nil {
			return err
		}

		if len(newVars) > 0 {
			// Acquire a lock since this function is called in parallel
			l.Lock()
			defer l.Unlock()

			// Update variables
			for k, v := range newVars {
				vars[k] = v
			}
		}

		return nil
	}
}

func (t *terraformProvider) init(vars map[string]string) (err error) {
	t.Once.Do(func() {
		var rc *ResourceConfig
		if t.Config != nil {
			if err := t.Config.RawConfig.Interpolate(vars); err != nil {
				panic(err)
			}

			rc = &ResourceConfig{
				ComputedKeys: t.Config.RawConfig.UnknownKeys(),
				Raw:          t.Config.RawConfig.Config(),
			}
		}

		err = t.Provider.Configure(rc)
	})

	return
}
