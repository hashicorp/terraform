package terraform

import (
	"fmt"
	"sort"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
)

// InputMode defines what sort of input will be asked for when Input
// is called on Context.
type InputMode byte

const (
	// InputModeVar asks for variables
	InputModeVar InputMode = 1 << iota

	// InputModeProvider asks for provider variables
	InputModeProvider

	// InputModeStd is the standard operating mode and asks for both variables
	// and providers.
	InputModeStd = InputModeVar | InputModeProvider
)

// ContextOpts are the user-configurable options to create a context with
// NewContext.
type ContextOpts struct {
	Diff         *Diff
	Hooks        []Hook
	Module       *module.Tree
	Parallelism  int
	State        *State
	Providers    map[string]ResourceProviderFactory
	Provisioners map[string]ResourceProvisionerFactory
	Variables    map[string]string

	UIInput UIInput
}

// Context represents all the context that Terraform needs in order to
// perform operations on infrastructure. This structure is built using
// NewContext. See the documentation for that.
type Context struct {
	diff         *Diff
	diffLock     sync.RWMutex
	hooks        []Hook
	module       *module.Tree
	providers    map[string]ResourceProviderFactory
	provisioners map[string]ResourceProvisionerFactory
	sh           *stopHook
	state        *State
	stateLock    sync.RWMutex
	uiInput      UIInput
	variables    map[string]string

	l                   sync.Mutex // Lock acquired during any task
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

	return &Context{
		diff:                opts.Diff,
		hooks:               hooks,
		module:              opts.Module,
		providers:           opts.Providers,
		providerInputConfig: make(map[string]map[string]interface{}),
		provisioners:        opts.Provisioners,
		sh:                  sh,
		state:               state,
		uiInput:             opts.UIInput,
		variables:           opts.Variables,
	}
}

// Graph returns the graph for this config.
func (c *Context) Graph() (*Graph, error) {
	return c.GraphBuilder().Build(RootModulePath)
}

// GraphBuilder returns the GraphBuilder that will be used to create
// the graphs for this context.
func (c *Context) GraphBuilder() GraphBuilder {
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
			v := m[n]
			switch v.Type() {
			case config.VariableTypeMap:
				continue
			case config.VariableTypeString:
				// Good!
			default:
				panic(fmt.Sprintf("Unknown variable type: %#v", v.Type()))
			}

			var defaultString string
			if v.Default != nil {
				defaultString = v.Default.(string)
			}

			// Ask the user for a value for this variable
			var value string
			for {
				var err error
				value, err = c.uiInput.Input(&InputOpts{
					Id:          fmt.Sprintf("var.%s", n),
					Query:       fmt.Sprintf("var.%s", n),
					Default:     defaultString,
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
		// Do the walk
		if _, err := c.walk(walkInput); err != nil {
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
	c.state = c.state.deepcopy()

	// Do the walk
	_, err := c.walk(walkApply)

	// Clean out any unused things
	c.state.prune()
	println(fmt.Sprintf("%#v", c.state))

	return c.state, err
}

// Plan generates an execution plan for the given context.
//
// The execution plan encapsulates the context and can be stored
// in order to reinstantiate a context later for Apply.
//
// Plan also updates the diff of this context to be the diff generated
// by the plan, so Apply can be called after.
func (c *Context) Plan(opts *PlanOpts) (*Plan, error) {
	v := c.acquireRun()
	defer c.releaseRun(v)

	p := &Plan{
		Module: c.module,
		Vars:   c.variables,
		State:  c.state,
	}

	var operation walkOperation
	if opts != nil && opts.Destroy {
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
			c.state = old.deepcopy()
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

	// Do the walk
	if _, err := c.walk(operation); err != nil {
		return nil, err
	}
	p.Diff = c.diff

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
	c.state = c.state.deepcopy()

	// Do the walk
	if _, err := c.walk(walkRefresh); err != nil {
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

	// Walk
	walker, err := c.walk(walkValidate)
	if err != nil {
		return nil, multierror.Append(errs, err).Errors
	}

	// Return the result
	rerrs := multierror.Append(errs, walker.ValidationErrors...)
	return walker.ValidationWarnings, rerrs.Errors
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

func (c *Context) walk(operation walkOperation) (*ContextGraphWalker, error) {
	// Build the graph
	graph, err := c.GraphBuilder().Build(RootModulePath)
	if err != nil {
		return nil, err
	}

	// Walk the graph
	walker := &ContextGraphWalker{Context: c, Operation: operation}
	return walker, graph.Walk(walker)
}

// walkOperation is an enum which tells the walkContext what to do.
type walkOperation byte

const (
	walkInvalid walkOperation = iota
	walkInput
	walkApply
	walkPlan
	walkPlanDestroy
	walkRefresh
	walkValidate
)
