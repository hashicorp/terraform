package terraform

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
)

// InputMode defines what sort of input will be asked for when Input
// is called on Context.
type InputMode byte

const (
	// InputModeVar asks for all variables
	InputModeVar InputMode = 1 << iota

	// InputModeVarUnset asks for variables which are not set yet
	InputModeVarUnset

	// InputModeProvider asks for provider variables
	InputModeProvider

	// InputModeStd is the standard operating mode and asks for both variables
	// and providers.
	InputModeStd = InputModeVar | InputModeProvider
)

// ContextOpts are the user-configurable options to create a context with
// NewContext.
type ContextOpts struct {
	Destroy      bool
	Diff         *Diff
	Hooks        []Hook
	Module       *module.Tree
	Parallelism  int
	State        *State
	Providers    map[string]ResourceProviderFactory
	Provisioners map[string]ResourceProvisionerFactory
	Targets      []string
	Variables    map[string]string

	UIInput UIInput
}

// Context represents all the context that Terraform needs in order to
// perform operations on infrastructure. This structure is built using
// NewContext. See the documentation for that.
type Context struct {
	destroy      bool
	diff         *Diff
	diffLock     sync.RWMutex
	hooks        []Hook
	module       *module.Tree
	providers    map[string]ResourceProviderFactory
	provisioners map[string]ResourceProvisionerFactory
	sh           *stopHook
	state        *State
	stateLock    sync.RWMutex
	targets      []string
	uiInput      UIInput
	variables    map[string]string

	l                   sync.Mutex // Lock acquired during any task
	parallelSem         Semaphore
	providerInputConfig map[string]map[string]interface{}
	runCh               <-chan struct{}
}

// NewContext creates a new Context structure.
//
// Once a Context is creator, the pointer values within ContextOpts
// should not be mutated in any way, since the pointers are copied, not
// the values themselves.
func NewContext(opts *ContextOpts) *Context {
	// Copy all the hooks and add our stop hook. We don't append directly
	// to the Config so that we're not modifying that in-place.
	sh := new(stopHook)
	hooks := make([]Hook, len(opts.Hooks)+1)
	copy(hooks, opts.Hooks)
	hooks[len(opts.Hooks)] = sh

	state := opts.State
	if state == nil {
		state = new(State)
		state.init()
	}

	// Determine parallelism, default to 10. We do this both to limit
	// CPU pressure but also to have an extra guard against rate throttling
	// from providers.
	par := opts.Parallelism
	if par == 0 {
		par = 10
	}

	// Setup the variables. We first take the variables given to us.
	// We then merge in the variables set in the environment.
	variables := make(map[string]string)
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, VarEnvPrefix) {
			continue
		}

		// Strip off the prefix and get the value after the first "="
		idx := strings.Index(v, "=")
		k := v[len(VarEnvPrefix):idx]
		v = v[idx+1:]

		// Override the command-line set variable
		variables[k] = v
	}
	for k, v := range opts.Variables {
		variables[k] = v
	}

	return &Context{
		destroy:      opts.Destroy,
		diff:         opts.Diff,
		hooks:        hooks,
		module:       opts.Module,
		providers:    opts.Providers,
		provisioners: opts.Provisioners,
		state:        state,
		targets:      opts.Targets,
		uiInput:      opts.UIInput,
		variables:    variables,

		parallelSem:         NewSemaphore(par),
		providerInputConfig: make(map[string]map[string]interface{}),
		sh:                  sh,
	}
}

type ContextGraphOpts struct {
	Validate bool
	Verbose  bool
}

// Graph returns the graph for this config.
func (c *Context) Graph(g *ContextGraphOpts) (*Graph, error) {
	return c.graphBuilder(g).Build(RootModulePath)
}

// GraphBuilder returns the GraphBuilder that will be used to create
// the graphs for this context.
func (c *Context) graphBuilder(g *ContextGraphOpts) GraphBuilder {
	// TODO test
	providers := make([]string, 0, len(c.providers))
	for k, _ := range c.providers {
		providers = append(providers, k)
	}

	provisioners := make([]string, 0, len(c.provisioners))
	for k, _ := range c.provisioners {
		provisioners = append(provisioners, k)
	}

	return &BuiltinGraphBuilder{
		Root:         c.module,
		Diff:         c.diff,
		Providers:    providers,
		Provisioners: provisioners,
		State:        c.state,
		Targets:      c.targets,
		Destroy:      c.destroy,
		Validate:     g.Validate,
		Verbose:      g.Verbose,
	}
}

// Input asks for input to fill variables and provider configurations.
// This modifies the configuration in-place, so asking for Input twice
// may result in different UI output showing different current values.
func (c *Context) Input(mode InputMode) error {
	v := c.acquireRun()
	defer c.releaseRun(v)

	if mode&InputModeVar != 0 {
		// Walk the variables first for the root module. We walk them in
		// alphabetical order for UX reasons.
		rootConf := c.module.Config()
		names := make([]string, len(rootConf.Variables))
		m := make(map[string]*config.Variable)
		for i, v := range rootConf.Variables {
			names[i] = v.Name
			m[v.Name] = v
		}
		sort.Strings(names)
		for _, n := range names {
			// If we only care about unset variables, then if the variable
			// is set, continue on.
			if mode&InputModeVarUnset != 0 {
				if _, ok := c.variables[n]; ok {
					continue
				}
			}

			v := m[n]
			switch v.Type() {
			case config.VariableTypeUnknown:
				continue
			case config.VariableTypeMap:
				continue
			case config.VariableTypeString:
				// Good!
			default:
				panic(fmt.Sprintf("Unknown variable type: %#v", v.Type()))
			}

			// If the variable is not already set, and the variable defines a
			// default, use that for the value.
			if _, ok := c.variables[n]; !ok {
				if v.Default != nil {
					c.variables[n] = v.Default.(string)
					continue
				}
			}

			// Ask the user for a value for this variable
			var value string
			for {
				var err error
				value, err = c.uiInput.Input(&InputOpts{
					Id:          fmt.Sprintf("var.%s", n),
					Query:       fmt.Sprintf("var.%s", n),
					Description: v.Description,
				})
				if err != nil {
					return fmt.Errorf(
						"Error asking for %s: %s", n, err)
				}

				if value == "" && v.Required() {
					// Redo if it is required.
					continue
				}

				if value == "" {
					// No value, just exit the loop. With no value, we just
					// use whatever is currently set in variables.
					break
				}

				break
			}

			if value != "" {
				c.variables[n] = value
			}
		}
	}

	if mode&InputModeProvider != 0 {
		// Build the graph
		graph, err := c.Graph(&ContextGraphOpts{Validate: true})
		if err != nil {
			return err
		}

		// Do the walk
		if _, err := c.walk(graph, walkInput); err != nil {
			return err
		}
	}

	return nil
}

// Apply applies the changes represented by this context and returns
// the resulting state.
//
// In addition to returning the resulting state, this context is updated
// with the latest state.
func (c *Context) Apply() (*State, error) {
	v := c.acquireRun()
	defer c.releaseRun(v)

	// Copy our own state
	c.state = c.state.DeepCopy()

	// Build the graph
	graph, err := c.Graph(&ContextGraphOpts{Validate: true})
	if err != nil {
		return nil, err
	}

	// Do the walk
	if c.destroy {
		_, err = c.walk(graph, walkDestroy)
	} else {
		_, err = c.walk(graph, walkApply)
	}

	// Clean out any unused things
	c.state.prune()

	return c.state, err
}

// Plan generates an execution plan for the given context.
//
// The execution plan encapsulates the context and can be stored
// in order to reinstantiate a context later for Apply.
//
// Plan also updates the diff of this context to be the diff generated
// by the plan, so Apply can be called after.
func (c *Context) Plan() (*Plan, error) {
	v := c.acquireRun()
	defer c.releaseRun(v)

	p := &Plan{
		Module:  c.module,
		Vars:    c.variables,
		State:   c.state,
		Targets: c.targets,
	}

	var operation walkOperation
	if c.destroy {
		operation = walkPlanDestroy
	} else {
		// Set our state to be something temporary. We do this so that
		// the plan can update a fake state so that variables work, then
		// we replace it back with our old state.
		old := c.state
		if old == nil {
			c.state = &State{}
			c.state.init()
		} else {
			c.state = old.DeepCopy()
		}
		defer func() {
			c.state = old
		}()

		operation = walkPlan
	}

	// Setup our diff
	c.diffLock.Lock()
	c.diff = new(Diff)
	c.diff.init()
	c.diffLock.Unlock()

	// Build the graph
	graph, err := c.Graph(&ContextGraphOpts{Validate: true})
	if err != nil {
		return nil, err
	}

	// Do the walk
	if _, err := c.walk(graph, operation); err != nil {
		return nil, err
	}
	p.Diff = c.diff

	// Now that we have a diff, we can build the exact graph that Apply will use
	// and catch any possible cycles during the Plan phase.
	if _, err := c.Graph(&ContextGraphOpts{Validate: true}); err != nil {
		return nil, err
	}

	return p, nil
}

// Refresh goes through all the resources in the state and refreshes them
// to their latest state. This will update the state that this context
// works with, along with returning it.
//
// Even in the case an error is returned, the state will be returned and
// will potentially be partially updated.
func (c *Context) Refresh() (*State, error) {
	v := c.acquireRun()
	defer c.releaseRun(v)

	// Copy our own state
	c.state = c.state.DeepCopy()

	// Build the graph
	graph, err := c.Graph(&ContextGraphOpts{Validate: true})
	if err != nil {
		return nil, err
	}

	// Do the walk
	if _, err := c.walk(graph, walkRefresh); err != nil {
		return nil, err
	}

	// Clean out any unused things
	c.state.prune()

	return c.state, nil
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
	v := c.acquireRun()
	defer c.releaseRun(v)

	var errs error

	// Validate the configuration itself
	if err := c.module.Validate(); err != nil {
		errs = multierror.Append(errs, err)
	}

	// This only needs to be done for the root module, since inter-module
	// variables are validated in the module tree.
	if config := c.module.Config(); config != nil {
		// Validate the user variables
		if err := smcUserVariables(config, c.variables); len(err) > 0 {
			errs = multierror.Append(errs, err...)
		}
	}

	// If we have errors at this point, the graphing has no chance,
	// so just bail early.
	if errs != nil {
		return nil, []error{errs}
	}

	// Build the graph so we can walk it and run Validate on nodes.
	// We also validate the graph generated here, but this graph doesn't
	// necessarily match the graph that Plan will generate, so we'll validate the
	// graph again later after Planning.
	graph, err := c.Graph(&ContextGraphOpts{Validate: true})
	if err != nil {
		return nil, []error{err}
	}

	// Walk
	walker, err := c.walk(graph, walkValidate)
	if err != nil {
		return nil, multierror.Append(errs, err).Errors
	}

	// Return the result
	rerrs := multierror.Append(errs, walker.ValidationErrors...)
	return walker.ValidationWarnings, rerrs.Errors
}

// Module returns the module tree associated with this context.
func (c *Context) Module() *module.Tree {
	return c.module
}

// Variables will return the mapping of variables that were defined
// for this Context. If Input was called, this mapping may be different
// than what was given.
func (c *Context) Variables() map[string]string {
	return c.variables
}

// SetVariable sets a variable after a context has already been built.
func (c *Context) SetVariable(k, v string) {
	c.variables[k] = v
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

func (c *Context) walk(
	graph *Graph, operation walkOperation) (*ContextGraphWalker, error) {
	// Walk the graph
	log.Printf("[DEBUG] Starting graph walk: %s", operation.String())
	walker := &ContextGraphWalker{Context: c, Operation: operation}
	return walker, graph.Walk(walker)
}
