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

// This is a function type used to implement a walker for the resource
// tree internally on the Terraform structure.
type genericWalkFunc func(*Resource) (map[string]string, error)

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

	l     sync.Mutex
	cond  *sync.Cond
	runCh <-chan struct{}
	sh    *stopHook
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
	sh := new(stopHook)

	// Copy all the hooks and add our stop hook. We don't append directly
	// to the Config so that we're not modifying that in-place.
	hooks := make([]Hook, len(opts.Hooks)+1)
	copy(hooks, opts.Hooks)
	hooks[len(opts.Hooks)] = sh

	return &Context{
		config:    opts.Config,
		diff:      opts.Diff,
		hooks:     hooks,
		state:     opts.State,
		providers: opts.Providers,
		variables: opts.Variables,

		cond: sync.NewCond(new(sync.Mutex)),
		sh:   sh,
	}
}

// Apply applies the changes represented by this context and returns
// the resulting state.
//
// In addition to returning the resulting state, this context is updated
// with the latest state.
func (c *Context) Apply() (*State, error) {
	v := c.acquireRun()
	defer c.releaseRun(v)

	g, err := Graph(&GraphOpts{
		Config:    c.config,
		Diff:      c.diff,
		Providers: c.providers,
		State:     c.state,
	})
	if err != nil {
		return nil, err
	}

	// Create our result. Make sure we preserve the prior states
	s := new(State)
	s.init()
	if c.state != nil {
		for k, v := range c.state.Resources {
			s.Resources[k] = v
		}
	}

	// Walk
	err = g.Walk(c.applyWalkFn(s))

	// Update our state, even if we have an error, for partial updates
	c.state = s

	return s, err
}

// Plan generates an execution plan for the given context.
//
// The execution plan encapsulates the context and can be stored
// in order to reinstantiate a context later for Apply.
//
// Plan also updates the diff of this context to be the diff generated
// by the plan, so Apply can be called after.
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

	// Update the diff so that our context is up-to-date
	c.diff = p.Diff

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

// Stop stops the running task.
//
// Stop will block until the task completes.
func (c *Context) Stop() {
	c.l.Lock()
	ch := c.runCh

	// If we aren't running, then just return
	if ch == nil {
		c.l.Unlock()
		return
	}

	// Tell the hook we want to stop
	c.sh.Stop()

	// Wait for us to stop
	c.l.Unlock()
	<-ch
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

func (c *Context) acquireRun() chan<- struct{} {
	c.l.Lock()
	defer c.l.Unlock()

	// Wait for no channel to exist
	for c.runCh != nil {
		c.l.Unlock()
		ch := c.runCh
		<-ch
		c.l.Lock()
	}

	ch := make(chan struct{})
	c.runCh = ch
	return ch
}

func (c *Context) releaseRun(ch chan<- struct{}) {
	c.l.Lock()
	defer c.l.Unlock()

	close(ch)
	c.runCh = nil
	c.sh.Reset()
}

func (c *Context) applyWalkFn(result *State) depgraph.WalkFunc {
	var l sync.Mutex

	// Initialize the result
	result.init()

	cb := func(r *Resource) (map[string]string, error) {
		diff := r.Diff
		if diff.Empty() {
			return r.Vars(), nil
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

		for _, h := range c.hooks {
			handleHook(h.PreApply(r.Id, r.State, diff))
		}

		// With the completed diff, apply!
		log.Printf("[DEBUG] %s: Executing Apply", r.Id)
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
		if rs.ID == "" {
			delete(result.Resources, r.Id)
		} else {
			result.Resources[r.Id] = rs
		}
		l.Unlock()

		// Update the state for the resource itself
		r.State = rs

		for _, h := range c.hooks {
			handleHook(h.PostApply(r.Id, r.State))
		}

		// Determine the new state and update variables
		err = nil
		if len(errs) > 0 {
			err = &multierror.Error{Errors: errs}
		}

		return r.Vars(), err
	}

	return c.genericWalkFn(c.variables, cb)
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
