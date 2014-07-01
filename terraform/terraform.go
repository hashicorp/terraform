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
	hooks     []Hook
	providers map[string]ResourceProviderFactory
}

// This is a function type used to implement a walker for the resource
// tree internally on the Terraform structure.
type genericWalkFunc func(*Resource) (map[string]string, error)

// Config is the configuration that must be given to instantiate
// a Terraform structure.
type Config struct {
	Hooks     []Hook
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
		hooks:     c.Hooks,
		providers: c.Providers,
	}, nil
}

func (t *Terraform) Apply(p *Plan) (*State, error) {
	// Make sure we're working with a plan that doesn't have null pointers
	// everywhere, and is instead just empty otherwise.
	p.init()

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

func (t *Terraform) Plan(opts *PlanOpts) (*Plan, error) {
	g, err := t.Graph(opts.Config, opts.State)
	if err != nil {
		return nil, err
	}

	return t.plan(g, opts)
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
	err := g.Walk(t.applyWalkFn(s, p))

	// Now that we've built the state and have the graph, re-calculate
	// the dependencies for our state based on what we did.

	return s, err
}

func (t *Terraform) plan(g *depgraph.Graph, opts *PlanOpts) (*Plan, error) {
	p := &Plan{
		Config: opts.Config,
		Vars:   opts.Vars,
		State:  opts.State,
	}
	err := g.Walk(t.planWalkFn(p, opts))
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
		for _, h := range t.hooks {
			// TODO: return value
			h.PreRefresh(r.Id, r.State)
		}

		rs, err := r.Provider.Refresh(r.State)
		if err != nil {
			return nil, err
		}
		if rs == nil {
			rs = new(ResourceState)
		}

		// Fix the type to be the type we have
		rs.Type = r.State.Type

		l.Lock()
		result.Resources[r.Id] = rs
		l.Unlock()

		for _, h := range t.hooks {
			// TODO: return value
			h.PostRefresh(r.Id, rs)
		}

		return nil, nil
	}

	return t.genericWalkFn(nil, cb)
}

func (t *Terraform) applyWalkFn(
	result *State,
	p *Plan) depgraph.WalkFunc {
	var l sync.Mutex

	// Initialize the result
	result.init()

	cb := func(r *Resource) (map[string]string, error) {
		diff, ok := p.Diff.Resources[r.Id]
		if !ok {
			// Skip if there is no diff for a resource
			log.Printf("[DEBUG] No diff for %s, skipping.", r.Id)
			return nil, nil
		}

		if !diff.Destroy {
			var err error
			diff, err = r.Provider.Diff(r.State, r.Config)
			if err != nil {
				return nil, err
			}
		}

		// TODO(mitchellh): we need to verify the diff doesn't change
		// anything and that the diff has no computed values (pre-computed)

		for _, h := range t.hooks {
			// TODO: return value
			h.PreApply(r.Id, r.State, diff)
		}

		// With the completed diff, apply!
		rs, err := r.Provider.Apply(r.State, diff)
		if err != nil {
			return nil, err
		}

		// Make sure the result is instantiated
		if rs == nil {
			rs = new(ResourceState)
		}

		// Force the resource state type to be our type
		rs.Type = r.State.Type

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

		for _, h := range t.hooks {
			// TODO: return value
			h.PostApply(r.Id, r.State)
		}

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

	return t.genericWalkFn(p.Vars, cb)
}

func (t *Terraform) planWalkFn(result *Plan, opts *PlanOpts) depgraph.WalkFunc {
	var l sync.Mutex

	// Initialize the result
	result.init()

	cb := func(r *Resource) (map[string]string, error) {
		var diff *ResourceDiff

		for _, h := range t.hooks {
			// TODO: return value
			h.PreDiff(r.Id, r.State)
		}

		if opts.Destroy {
			if r.State.ID != "" {
				log.Printf("[DEBUG] %s: Making for destroy", r.Id)
				diff = &ResourceDiff{Destroy: true}
			}
		} else if r.Config == nil {
			log.Printf("[DEBUG] %s: Orphan, marking for destroy", r.Id)

			// This is an orphan (no config), so we mark it to be destroyed
			diff = &ResourceDiff{Destroy: true}
		} else {
			log.Printf("[DEBUG] %s: Executing diff", r.Id)

			// Get a diff from the newest state
			var err error
			diff, err = r.Provider.Diff(r.State, r.Config)
			if err != nil {
				return nil, err
			}
		}

		l.Lock()
		if !diff.Empty() {
			result.Diff.Resources[r.Id] = diff
		}
		l.Unlock()

		for _, h := range t.hooks {
			// TODO: return value
			h.PostDiff(r.Id, diff)
		}

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

	return t.genericWalkFn(opts.Vars, cb)
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
				log.Printf("[INFO] Configuring provider: %s", k)
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
		if !rn.Orphan {
			if rn.Resource.Config == nil {
				if rn.Config == nil {
					rn.Resource.Config = new(ResourceConfig)
				} else {
					rn.Resource.Config = NewResourceConfig(rn.Config.RawConfig)
				}
			}
		} else {
			rn.Resource.Config = nil
		}

		// Call the callack
		log.Printf("[INFO] Walking: %s", rn.Resource.Id)
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
