package terraform

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/depgraph"
	"github.com/hashicorp/terraform/helper/multierror"
)

// Context represents all the context that Terraform needs in order to
// perform operations on infrastructure. This structure is built using
// ContextOpts and NewContext. See the documentation for those.
//
// Additionally, a context can be created from a Plan using Plan.Context.
type Context struct {
	config    *config.Config
	diff      *Diff
	hooks     []Hook
	state     *State
	providers map[string]ResourceProviderFactory
	variables map[string]string
}

// ContextOpts are the user-creatable configuration structure to create
// a context with NewContext.
type ContextOpts struct {
	Config    *config.Config
	Diff      *Diff
	Hooks     []Hook
	State     *State
	Providers map[string]ResourceProviderFactory
	Variables map[string]string
}

// NewContext creates a new context.
//
// Once a context is created, the pointer values within ContextOpts should
// not be mutated in any way, since the pointers are copied, not the values
// themselves.
func NewContext(opts *ContextOpts) *Context {
	return &Context{
		config:    opts.Config,
		diff:      opts.Diff,
		hooks:     opts.Hooks,
		state:     opts.State,
		providers: opts.Providers,
		variables: opts.Variables,
	}
}

// Plan generates an execution plan for the given context.
//
// The execution plan encapsulates the context and can be stored
// in order to reinstantiate a context later for Apply.
func (c *Context) Plan(opts *PlanOpts) (*Plan, error) {
	g, err := Graph(&GraphOpts{
		Config:    c.config,
		Providers: c.providers,
		State:     c.state,
	})
	if err != nil {
		return nil, err
	}

	p := &Plan{
		Config: c.config,
		Vars:   c.variables,
		State:  c.state,
	}
	err = g.Walk(c.planWalkFn(p, opts))
	return p, err
}

// Refresh goes through all the resources in the state and refreshes them
// to their latest state. This will update the state that this context
// works with, along with returning it.
//
// Even in the case an error is returned, the state will be returned and
// will potentially be partially updated.
func (c *Context) Refresh() (*State, error) {
	g, err := Graph(&GraphOpts{
		Config:    c.config,
		Providers: c.providers,
		State:     c.state,
	})
	if err != nil {
		return c.state, err
	}

	s := new(State)
	s.init()
	err = g.Walk(c.refreshWalkFn(s))
	return s, err
}

// Validate validates the configuration and returns any warnings or errors.
func (c *Context) Validate() ([]string, []error) {
	var rerr *multierror.Error

	// Validate the configuration itself
	if err := c.config.Validate(); err != nil {
		rerr = multierror.ErrorAppend(rerr, err)
	}

	// Validate the user variables
	if errs := smcUserVariables(c.config, c.variables); len(errs) > 0 {
		rerr = multierror.ErrorAppend(rerr, errs...)
	}

	var errs []error
	if rerr != nil && len(rerr.Errors) > 0 {
		errs = rerr.Errors
	}

	return nil, errs
}

func (c *Context) planWalkFn(result *Plan, opts *PlanOpts) depgraph.WalkFunc {
	var l sync.Mutex

	// If we were given nil options, instantiate it
	if opts == nil {
		opts = new(PlanOpts)
	}

	// Initialize the result
	result.init()

	cb := func(r *Resource) (map[string]string, error) {
		var diff *ResourceDiff

		for _, h := range c.hooks {
			handleHook(h.PreDiff(r.Id, r.State))
		}

		if opts.Destroy {
			if r.State.ID != "" {
				log.Printf("[DEBUG] %s: Making for destroy", r.Id)
				diff = &ResourceDiff{Destroy: true}
			} else {
				log.Printf("[DEBUG] %s: Not marking for destroy, no ID", r.Id)
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

		for _, h := range c.hooks {
			handleHook(h.PostDiff(r.Id, diff))
		}

		// Determine the new state and update variables
		if !diff.Empty() {
			r.State = r.State.MergeDiff(diff)
		}

		return r.Vars(), nil
	}

	return c.genericWalkFn(c.variables, cb)
}

func (c *Context) refreshWalkFn(result *State) depgraph.WalkFunc {
	var l sync.Mutex

	cb := func(r *Resource) (map[string]string, error) {
		for _, h := range c.hooks {
			handleHook(h.PreRefresh(r.Id, r.State))
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

		for _, h := range c.hooks {
			handleHook(h.PostRefresh(r.Id, rs))
		}

		return nil, nil
	}

	return c.genericWalkFn(c.variables, cb)
}

func (c *Context) genericWalkFn(
	invars map[string]string,
	cb genericWalkFunc) depgraph.WalkFunc {
	var l sync.RWMutex

	// Initialize the variables for application
	vars := make(map[string]string)
	for k, v := range invars {
		vars[fmt.Sprintf("var.%s", k)] = v
	}

	// This will keep track of whether we're stopped or not
	var stop uint32 = 0

	return func(n *depgraph.Noun) error {
		// If it is the root node, ignore
		if n.Name == GraphRootNode {
			return nil
		}

		// If we're stopped, return right away
		if atomic.LoadUint32(&stop) != 0 {
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

		l.RLock()
		if len(vars) > 0 && rn.Config != nil {
			if err := rn.Config.RawConfig.Interpolate(vars); err != nil {
				panic(fmt.Sprintf("Interpolate error: %s", err))
			}

			// Force the config to be set later
			rn.Resource.Config = nil
		}
		l.RUnlock()

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

		// Handle recovery of special panic scenarios
		defer func() {
			if v := recover(); v != nil {
				if v == HookActionHalt {
					atomic.StoreUint32(&stop, 1)
				} else {
					panic(v)
				}
			}
		}()

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
