package terraform

import (
	"fmt"
	"log"
	"sync"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/depgraph"
)

// Terraform is the primary structure that is used to interact with
// Terraform from code, and can perform operations such as returning
// all resources, a resource tree, a specific resource, etc.
type Terraform struct {
	providers map[string]ResourceProviderFactory
}

// This is a function type used to implement a walker for the resource
// tree internally on the Terraform structure.
type genericWalkFunc func(*Resource) (map[string]string, error)

// Config is the configuration that must be given to instantiate
// a Terraform structure.
type Config struct {
	Providers map[string]ResourceProviderFactory
}

// New creates a new Terraform structure, initializes resource providers
// for the given configuration, etc.
//
// Semantic checks of the entire configuration structure are done at this
// time, as well as richer checks such as verifying that the resource providers
// can be properly initialized, can be configured, etc.
func New(c *Config) (*Terraform, error) {
	return &Terraform{
		providers: c.Providers,
	}, nil
}

func (t *Terraform) Apply(p *Plan) (*State, error) {
	g, err := t.Graph(p.Config, p.State)
	if err != nil {
		return nil, err
	}

	return t.apply(g, p)
}

// Graph returns the dependency graph for the given configuration and
// state file.
//
// The resulting graph may have more resources than the configuration, because
// it can contain resources in the state file that need to be modified.
func (t *Terraform) Graph(c *config.Config, s *State) (*depgraph.Graph, error) {
	// Get the basic graph with the raw metadata
	g := Graph(c, s)
	if err := g.Validate(); err != nil {
		return nil, err
	}

	// Fill the graph with the providers
	if err := GraphFull(g, t.providers); err != nil {
		return nil, err
	}

	// Validate the graph so that it can setup a root and such
	if err := g.Validate(); err != nil {
		return nil, err
	}

	return g, nil
}

func (t *Terraform) Plan(
	c *config.Config, s *State, vs map[string]string) (*Plan, error) {
	g, err := t.Graph(c, s)
	if err != nil {
		return nil, err
	}

	return t.plan(g, c, s, vs)
}

// Refresh goes through all the resources in the state and refreshes them
// to their latest status.
func (t *Terraform) Refresh(c *config.Config, s *State) (*State, error) {
	g, err := t.Graph(c, s)
	if err != nil {
		return s, err
	}

	return t.refresh(g)
}

func (t *Terraform) apply(
	g *depgraph.Graph,
	p *Plan) (*State, error) {
	s := new(State)
	err := g.Walk(t.applyWalkFn(s, p.Vars))
	return s, err
}

func (t *Terraform) plan(
	g *depgraph.Graph,
	c *config.Config,
	s *State,
	vs map[string]string) (*Plan, error) {
	p := &Plan{
		Config: c,
		Vars:   vs,
		State:  s,
	}
	err := g.Walk(t.planWalkFn(p, vs))
	return p, err
}

func (t *Terraform) refresh(g *depgraph.Graph) (*State, error) {
	s := new(State)
	err := g.Walk(t.refreshWalkFn(s))
	return s, err
}

func (t *Terraform) refreshWalkFn(result *State) depgraph.WalkFunc {
	var l sync.Mutex

	// Initialize the result so we don't have to nil check everywhere
	result.init()

	cb := func(r *Resource) (map[string]string, error) {
		rs, err := r.Provider.Refresh(r.State)
		if err != nil {
			return nil, err
		}

		// Fix the type to be the type we have
		rs.Type = r.State.Type

		l.Lock()
		result.Resources[r.Id] = rs
		l.Unlock()

		return nil, nil
	}

	return t.genericWalkFn(nil, cb)
}

func (t *Terraform) applyWalkFn(
	result *State,
	vs map[string]string) depgraph.WalkFunc {
	var l sync.Mutex

	// Initialize the result
	result.init()

	cb := func(r *Resource) (map[string]string, error) {
		// Get the latest diff since there are no computed values anymore
		diff, err := r.Provider.Diff(r.State, r.Config)
		if err != nil {
			return nil, err
		}

		// TODO(mitchellh): we need to verify the diff doesn't change
		// anything and that the diff has no computed values (pre-computed)

		// With the completed diff, apply!
		rs, err := r.Provider.Apply(r.State, diff)
		if err != nil {
			return nil, err
		}

		// If no state was returned, then no variables were updated so
		// just return.
		if rs == nil {
			return nil, nil
		}

		var errs []error
		for ak, av := range rs.Attributes {
			// If the value is the unknown variable value, then it is an error.
			// In this case we record the error and remove it from the state
			if av == config.UnknownVariableValue {
				errs = append(errs, fmt.Errorf(
					"Attribute with unknown value: %s", ak))
				delete(rs.Attributes, ak)
			}
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

		err = nil
		if len(errs) > 0 {
			err = &MultiError{Errors: errs}
		}

		return vars, err
	}

	return t.genericWalkFn(vs, cb)
}

func (t *Terraform) planWalkFn(
	result *Plan, vs map[string]string) depgraph.WalkFunc {
	var l sync.Mutex

	// Initialize the result
	result.init()

	cb := func(r *Resource) (map[string]string, error) {
		// Get a diff from the newest state
		diff, err := r.Provider.Diff(r.State, r.Config)
		if err != nil {
			return nil, err
		}

		l.Lock()
		if !diff.Empty() {
			result.Diff.Resources[r.Id] = diff
		}
		l.Unlock()

		// Determine the new state and update variables
		vars := make(map[string]string)
		if !diff.Empty() {
			r.State = r.State.MergeDiff(diff)
		}
		if r.State != nil {
			for ak, av := range r.State.Attributes {
				vars[fmt.Sprintf("%s.%s", r.Id, ak)] = av
			}
		}

		return vars, nil
	}

	return t.genericWalkFn(vs, cb)
}

func (t *Terraform) genericWalkFn(
	invars map[string]string,
	cb genericWalkFunc) depgraph.WalkFunc {
	var l sync.Mutex

	// Initialize the variables for application
	vars := make(map[string]string)
	for k, v := range invars {
		vars[fmt.Sprintf("var.%s", k)] = v
	}

	return func(n *depgraph.Noun) error {
		// If it is the root node, ignore
		if n.Name == GraphRootNode {
			return nil
		}

		switch m := n.Meta.(type) {
		case *GraphNodeResource:
		case *GraphNodeResourceProvider:
			var rc *ResourceConfig
			if m.Config != nil {
				if err := m.Config.RawConfig.Interpolate(vars); err != nil {
					panic(err)
				}
				rc = NewResourceConfig(m.Config.RawConfig)
			}

			for k, p := range m.Providers {
				log.Printf("Configuring provider: %s", k)
				err := p.Configure(rc)
				if err != nil {
					return err
				}
			}

			return nil
		}

		rn := n.Meta.(*GraphNodeResource)
		if len(vars) > 0 && rn.Config != nil {
			if err := rn.Config.RawConfig.Interpolate(vars); err != nil {
				panic(fmt.Sprintf("Interpolate error: %s", err))
			}

			// Force the config to be set later
			rn.Resource.Config = nil
		}

		// Make sure that at least some resource configuration is set
		if rn.Resource.Config == nil {
			if rn.Config == nil {
				rn.Resource.Config = new(ResourceConfig)
			} else {
				rn.Resource.Config = NewResourceConfig(rn.Config.RawConfig)
			}
		}

		// Call the callack
		newVars, err := cb(rn.Resource)
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
