package terraform

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
)

// InputMode defines what sort of input will be asked for when Input
// is called on Context.
type InputMode byte

const (
	// InputModeVar asks for all variables
	InputModeVar InputMode = 1 << iota

	// InputModeVarUnset asks for variables which are not set yet.
	// InputModeVar must be set for this to have an effect.
	InputModeVarUnset

	// InputModeProvider asks for provider variables
	InputModeProvider

	// InputModeStd is the standard operating mode and asks for both variables
	// and providers.
	InputModeStd = InputModeVar | InputModeProvider
)

var (
	// contextFailOnShadowError will cause Context operations to return
	// errors when shadow operations fail. This is only used for testing.
	contextFailOnShadowError = false

	// contextTestDeepCopyOnPlan will perform a Diff DeepCopy on every
	// Plan operation, effectively testing the Diff DeepCopy whenever
	// a Plan occurs. This is enabled for tests.
	contextTestDeepCopyOnPlan = false
)

// ContextOpts are the user-configurable options to create a context with
// NewContext.
type ContextOpts struct {
	Meta               *ContextMeta
	Destroy            bool
	Diff               *Diff
	Hooks              []Hook
	Module             *module.Tree
	Parallelism        int
	State              *State
	StateFutureAllowed bool
	ProviderResolver   ResourceProviderResolver
	Provisioners       map[string]ResourceProvisionerFactory
	Shadow             bool
	Targets            []string
	Variables          map[string]interface{}

	// If non-nil, will apply as additional constraints on the provider
	// plugins that will be requested from the provider resolver.
	ProviderSHA256s    map[string][]byte
	SkipProviderVerify bool

	UIInput UIInput
}

// ContextMeta is metadata about the running context. This is information
// that this package or structure cannot determine on its own but exposes
// into Terraform in various ways. This must be provided by the Context
// initializer.
type ContextMeta struct {
	Env string // Env is the state environment
}

// Context represents all the context that Terraform needs in order to
// perform operations on infrastructure. This structure is built using
// NewContext. See the documentation for that.
//
// Extra functions on Context can be found in context_*.go files.
type Context struct {
	// Maintainer note: Anytime this struct is changed, please verify
	// that newShadowContext still does the right thing. Tests should
	// fail regardless but putting this note here as well.

	components contextComponentFactory
	destroy    bool
	diff       *Diff
	diffLock   sync.RWMutex
	hooks      []Hook
	meta       *ContextMeta
	module     *module.Tree
	sh         *stopHook
	shadow     bool
	state      *State
	stateLock  sync.RWMutex
	targets    []string
	uiInput    UIInput
	variables  map[string]interface{}

	l                   sync.Mutex // Lock acquired during any task
	parallelSem         Semaphore
	providerInputConfig map[string]map[string]interface{}
	providerSHA256s     map[string][]byte
	runLock             sync.Mutex
	runCond             *sync.Cond
	runContext          context.Context
	runContextCancel    context.CancelFunc
	shadowErr           error
}

// NewContext creates a new Context structure.
//
// Once a Context is creator, the pointer values within ContextOpts
// should not be mutated in any way, since the pointers are copied, not
// the values themselves.
func NewContext(opts *ContextOpts) (*Context, error) {
	// Validate the version requirement if it is given
	if opts.Module != nil {
		if err := CheckRequiredVersion(opts.Module); err != nil {
			return nil, err
		}
	}

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

	// If our state is from the future, then error. Callers can avoid
	// this error by explicitly setting `StateFutureAllowed`.
	if !opts.StateFutureAllowed && state.FromFutureTerraform() {
		return nil, fmt.Errorf(
			"Terraform doesn't allow running any operations against a state\n"+
				"that was written by a future Terraform version. The state is\n"+
				"reporting it is written by Terraform '%s'.\n\n"+
				"Please run at least that version of Terraform to continue.",
			state.TFVersion)
	}

	// Explicitly reset our state version to our current version so that
	// any operations we do will write out that our latest version
	// has run.
	state.TFVersion = Version

	// Determine parallelism, default to 10. We do this both to limit
	// CPU pressure but also to have an extra guard against rate throttling
	// from providers.
	par := opts.Parallelism
	if par == 0 {
		par = 10
	}

	// Set up the variables in the following sequence:
	//    0 - Take default values from the configuration
	//    1 - Take values from TF_VAR_x environment variables
	//    2 - Take values specified in -var flags, overriding values
	//        set by environment variables if necessary. This includes
	//        values taken from -var-file in addition.
	variables := make(map[string]interface{})
	if opts.Module != nil {
		var err error
		variables, err = Variables(opts.Module, opts.Variables)
		if err != nil {
			return nil, err
		}
	}

	// Bind available provider plugins to the constraints in config
	var providers map[string]ResourceProviderFactory
	if opts.ProviderResolver != nil {
		var err error
		deps := ModuleTreeDependencies(opts.Module, state)
		reqd := deps.AllPluginRequirements()
		if opts.ProviderSHA256s != nil && !opts.SkipProviderVerify {
			reqd.LockExecutables(opts.ProviderSHA256s)
		}
		providers, err = resourceProviderFactories(opts.ProviderResolver, reqd)
		if err != nil {
			return nil, err
		}
	} else {
		providers = make(map[string]ResourceProviderFactory)
	}

	diff := opts.Diff
	if diff == nil {
		diff = &Diff{}
	}

	return &Context{
		components: &basicComponentFactory{
			providers:    providers,
			provisioners: opts.Provisioners,
		},
		destroy:   opts.Destroy,
		diff:      diff,
		hooks:     hooks,
		meta:      opts.Meta,
		module:    opts.Module,
		shadow:    opts.Shadow,
		state:     state,
		targets:   opts.Targets,
		uiInput:   opts.UIInput,
		variables: variables,

		parallelSem:         NewSemaphore(par),
		providerInputConfig: make(map[string]map[string]interface{}),
		providerSHA256s:     opts.ProviderSHA256s,
		sh:                  sh,
	}, nil
}

type ContextGraphOpts struct {
	// If true, validates the graph structure (checks for cycles).
	Validate bool

	// Legacy graphs only: won't prune the graph
	Verbose bool
}

// Graph returns the graph used for the given operation type.
//
// The most extensive or complex graph type is GraphTypePlan.
func (c *Context) Graph(typ GraphType, opts *ContextGraphOpts) (*Graph, error) {
	if opts == nil {
		opts = &ContextGraphOpts{Validate: true}
	}

	log.Printf("[INFO] terraform: building graph: %s", typ)
	switch typ {
	case GraphTypeApply:
		return (&ApplyGraphBuilder{
			Module:       c.module,
			Diff:         c.diff,
			State:        c.state,
			Providers:    c.components.ResourceProviders(),
			Provisioners: c.components.ResourceProvisioners(),
			Targets:      c.targets,
			Destroy:      c.destroy,
			Validate:     opts.Validate,
		}).Build(RootModulePath)

	case GraphTypeInput:
		// The input graph is just a slightly modified plan graph
		fallthrough
	case GraphTypeValidate:
		// The validate graph is just a slightly modified plan graph
		fallthrough
	case GraphTypePlan:
		// Create the plan graph builder
		p := &PlanGraphBuilder{
			Module:    c.module,
			State:     c.state,
			Providers: c.components.ResourceProviders(),
			Targets:   c.targets,
			Validate:  opts.Validate,
		}

		// Some special cases for other graph types shared with plan currently
		var b GraphBuilder = p
		switch typ {
		case GraphTypeInput:
			b = InputGraphBuilder(p)
		case GraphTypeValidate:
			// We need to set the provisioners so those can be validated
			p.Provisioners = c.components.ResourceProvisioners()

			b = ValidateGraphBuilder(p)
		}

		return b.Build(RootModulePath)

	case GraphTypePlanDestroy:
		return (&DestroyPlanGraphBuilder{
			Module:   c.module,
			State:    c.state,
			Targets:  c.targets,
			Validate: opts.Validate,
		}).Build(RootModulePath)

	case GraphTypeRefresh:
		return (&RefreshGraphBuilder{
			Module:    c.module,
			State:     c.state,
			Providers: c.components.ResourceProviders(),
			Targets:   c.targets,
			Validate:  opts.Validate,
		}).Build(RootModulePath)
	}

	return nil, fmt.Errorf("unknown graph type: %s", typ)
}

// ShadowError returns any errors caught during a shadow operation.
//
// A shadow operation is an operation run in parallel to a real operation
// that performs the same tasks using new logic on copied state. The results
// are compared to ensure that the new logic works the same as the old logic.
// The shadow never affects the real operation or return values.
//
// The result of the shadow operation are only available through this function
// call after a real operation is complete.
//
// For API consumers of Context, you can safely ignore this function
// completely if you have no interest in helping report experimental feature
// errors to Terraform maintainers. Otherwise, please call this function
// after every operation and report this to the user.
//
// IMPORTANT: Shadow errors are _never_ critical: they _never_ affect
// the real state or result of a real operation. They are purely informational
// to assist in future Terraform versions being more stable. Please message
// this effectively to the end user.
//
// This must be called only when no other operation is running (refresh,
// plan, etc.). The result can be used in parallel to any other operation
// running.
func (c *Context) ShadowError() error {
	return c.shadowErr
}

// State returns a copy of the current state associated with this context.
//
// This cannot safely be called in parallel with any other Context function.
func (c *Context) State() *State {
	return c.state.DeepCopy()
}

// Interpolater returns an Interpolater built on a copy of the state
// that can be used to test interpolation values.
func (c *Context) Interpolater() *Interpolater {
	var varLock sync.Mutex
	var stateLock sync.RWMutex
	return &Interpolater{
		Operation:          walkApply,
		Meta:               c.meta,
		Module:             c.module,
		State:              c.state.DeepCopy(),
		StateLock:          &stateLock,
		VariableValues:     c.variables,
		VariableValuesLock: &varLock,
	}
}

// Input asks for input to fill variables and provider configurations.
// This modifies the configuration in-place, so asking for Input twice
// may result in different UI output showing different current values.
func (c *Context) Input(mode InputMode) error {
	defer c.acquireRun("input")()

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

			var valueType config.VariableType

			v := m[n]
			switch valueType = v.Type(); valueType {
			case config.VariableTypeUnknown:
				continue
			case config.VariableTypeMap:
				// OK
			case config.VariableTypeList:
				// OK
			case config.VariableTypeString:
				// OK
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

			// this should only happen during tests
			if c.uiInput == nil {
				log.Println("[WARN] Content.uiInput is nil")
				continue
			}

			// Ask the user for a value for this variable
			var value string
			retry := 0
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
					// Redo if it is required, but abort if we keep getting
					// blank entries
					if retry > 2 {
						return fmt.Errorf("missing required value for %q", n)
					}
					retry++
					continue
				}

				break
			}

			// no value provided, so don't set the variable at all
			if value == "" {
				continue
			}

			decoded, err := parseVariableAsHCL(n, value, valueType)
			if err != nil {
				return err
			}

			if decoded != nil {
				c.variables[n] = decoded
			}
		}
	}

	if mode&InputModeProvider != 0 {
		// Build the graph
		graph, err := c.Graph(GraphTypeInput, nil)
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
// Even in the case an error is returned, the state may be returned and will
// potentially be partially updated.  In addition to returning the resulting
// state, this context is updated with the latest state.
//
// If the state is required after an error, the caller should call
// Context.State, rather than rely on the return value.
//
// TODO: Apply and Refresh should either always return a state, or rely on the
//       State() method. Currently the helper/resource testing framework relies
//       on the absence of a returned state to determine if Destroy can be
//       called, so that will need to be refactored before this can be changed.
func (c *Context) Apply() (*State, error) {
	defer c.acquireRun("apply")()

	// Copy our own state
	c.state = c.state.DeepCopy()

	// Build the graph.
	graph, err := c.Graph(GraphTypeApply, nil)
	if err != nil {
		return nil, err
	}

	// Determine the operation
	operation := walkApply
	if c.destroy {
		operation = walkDestroy
	}

	// Walk the graph
	walker, err := c.walk(graph, operation)
	if len(walker.ValidationErrors) > 0 {
		err = multierror.Append(err, walker.ValidationErrors...)
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
	defer c.acquireRun("plan")()

	p := &Plan{
		Module:  c.module,
		Vars:    c.variables,
		State:   c.state,
		Targets: c.targets,

		TerraformVersion: VersionString(),
		ProviderSHA256s:  c.providerSHA256s,
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

	// Build the graph.
	graphType := GraphTypePlan
	if c.destroy {
		graphType = GraphTypePlanDestroy
	}
	graph, err := c.Graph(graphType, nil)
	if err != nil {
		return nil, err
	}

	// Do the walk
	walker, err := c.walk(graph, operation)
	if err != nil {
		return nil, err
	}
	p.Diff = c.diff

	// If this is true, it means we're running unit tests. In this case,
	// we perform a deep copy just to ensure that all context tests also
	// test that a diff is copy-able. This will panic if it fails. This
	// is enabled during unit tests.
	//
	// This should never be true during production usage, but even if it is,
	// it can't do any real harm.
	if contextTestDeepCopyOnPlan {
		p.Diff.DeepCopy()
	}

	/*
		// We don't do the reverification during the new destroy plan because
		// it will use a different apply process.
		if X_legacyGraph {
			// Now that we have a diff, we can build the exact graph that Apply will use
			// and catch any possible cycles during the Plan phase.
			if _, err := c.Graph(GraphTypeLegacy, nil); err != nil {
				return nil, err
			}
		}
	*/

	var errs error
	if len(walker.ValidationErrors) > 0 {
		errs = multierror.Append(errs, walker.ValidationErrors...)
	}
	return p, errs
}

// Refresh goes through all the resources in the state and refreshes them
// to their latest state. This will update the state that this context
// works with, along with returning it.
//
// Even in the case an error is returned, the state may be returned and
// will potentially be partially updated.
func (c *Context) Refresh() (*State, error) {
	defer c.acquireRun("refresh")()

	// Copy our own state
	c.state = c.state.DeepCopy()

	// Build the graph.
	graph, err := c.Graph(GraphTypeRefresh, nil)
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
	log.Printf("[WARN] terraform: Stop called, initiating interrupt sequence")

	c.l.Lock()
	defer c.l.Unlock()

	// If we're running, then stop
	if c.runContextCancel != nil {
		log.Printf("[WARN] terraform: run context exists, stopping")

		// Tell the hook we want to stop
		c.sh.Stop()

		// Stop the context
		c.runContextCancel()
		c.runContextCancel = nil
	}

	// Grab the condition var before we exit
	if cond := c.runCond; cond != nil {
		cond.Wait()
	}

	log.Printf("[WARN] terraform: stop complete")
}

// Validate validates the configuration and returns any warnings or errors.
func (c *Context) Validate() ([]string, []error) {
	defer c.acquireRun("validate")()

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
	graph, err := c.Graph(GraphTypeValidate, nil)
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

	sort.Strings(walker.ValidationWarnings)
	sort.Slice(rerrs.Errors, func(i, j int) bool {
		return rerrs.Errors[i].Error() < rerrs.Errors[j].Error()
	})

	return walker.ValidationWarnings, rerrs.Errors
}

// Module returns the module tree associated with this context.
func (c *Context) Module() *module.Tree {
	return c.module
}

// Variables will return the mapping of variables that were defined
// for this Context. If Input was called, this mapping may be different
// than what was given.
func (c *Context) Variables() map[string]interface{} {
	return c.variables
}

// SetVariable sets a variable after a context has already been built.
func (c *Context) SetVariable(k string, v interface{}) {
	c.variables[k] = v
}

func (c *Context) acquireRun(phase string) func() {
	// With the run lock held, grab the context lock to make changes
	// to the run context.
	c.l.Lock()
	defer c.l.Unlock()

	// Wait until we're no longer running
	for c.runCond != nil {
		c.runCond.Wait()
	}

	// Build our lock
	c.runCond = sync.NewCond(&c.l)

	// Setup debugging
	dbug.SetPhase(phase)

	// Create a new run context
	c.runContext, c.runContextCancel = context.WithCancel(context.Background())

	// Reset the stop hook so we're not stopped
	c.sh.Reset()

	// Reset the shadow errors
	c.shadowErr = nil

	return c.releaseRun
}

func (c *Context) releaseRun() {
	// Grab the context lock so that we can make modifications to fields
	c.l.Lock()
	defer c.l.Unlock()

	// setting the phase to "INVALID" lets us easily detect if we have
	// operations happening outside of a run, or we missed setting the proper
	// phase
	dbug.SetPhase("INVALID")

	// End our run. We check if runContext is non-nil because it can be
	// set to nil if it was cancelled via Stop()
	if c.runContextCancel != nil {
		c.runContextCancel()
	}

	// Unlock all waiting our condition
	cond := c.runCond
	c.runCond = nil
	cond.Broadcast()

	// Unset the context
	c.runContext = nil
}

func (c *Context) walk(graph *Graph, operation walkOperation) (*ContextGraphWalker, error) {
	// Keep track of the "real" context which is the context that does
	// the real work: talking to real providers, modifying real state, etc.
	realCtx := c

	log.Printf("[DEBUG] Starting graph walk: %s", operation.String())

	walker := &ContextGraphWalker{
		Context:     realCtx,
		Operation:   operation,
		StopContext: c.runContext,
	}

	// Watch for a stop so we can call the provider Stop() API.
	watchStop, watchWait := c.watchStop(walker)

	// Walk the real graph, this will block until it completes
	realErr := graph.Walk(walker)

	// Close the channel so the watcher stops, and wait for it to return.
	close(watchStop)
	<-watchWait

	return walker, realErr
}

// watchStop immediately returns a `stop` and a `wait` chan after dispatching
// the watchStop goroutine. This will watch the runContext for cancellation and
// stop the providers accordingly.  When the watch is no longer needed, the
// `stop` chan should be closed before waiting on the `wait` chan.
// The `wait` chan is important, because without synchronizing with the end of
// the watchStop goroutine, the runContext may also be closed during the select
// incorrectly causing providers to be stopped. Even if the graph walk is done
// at that point, stopping a provider permanently cancels its StopContext which
// can cause later actions to fail.
func (c *Context) watchStop(walker *ContextGraphWalker) (chan struct{}, <-chan struct{}) {
	stop := make(chan struct{})
	wait := make(chan struct{})

	// get the runContext cancellation channel now, because releaseRun will
	// write to the runContext field.
	done := c.runContext.Done()

	go func() {
		defer close(wait)
		// Wait for a stop or completion
		select {
		case <-done:
			// done means the context was canceled, so we need to try and stop
			// providers.
		case <-stop:
			// our own stop channel was closed.
			return
		}

		// If we're here, we're stopped, trigger the call.

		{
			// Copy the providers so that a misbehaved blocking Stop doesn't
			// completely hang Terraform.
			walker.providerLock.Lock()
			ps := make([]ResourceProvider, 0, len(walker.providerCache))
			for _, p := range walker.providerCache {
				ps = append(ps, p)
			}
			defer walker.providerLock.Unlock()

			for _, p := range ps {
				// We ignore the error for now since there isn't any reasonable
				// action to take if there is an error here, since the stop is still
				// advisory: Terraform will exit once the graph node completes.
				p.Stop()
			}
		}

		{
			// Call stop on all the provisioners
			walker.provisionerLock.Lock()
			ps := make([]ResourceProvisioner, 0, len(walker.provisionerCache))
			for _, p := range walker.provisionerCache {
				ps = append(ps, p)
			}
			defer walker.provisionerLock.Unlock()

			for _, p := range ps {
				// We ignore the error for now since there isn't any reasonable
				// action to take if there is an error here, since the stop is still
				// advisory: Terraform will exit once the graph node completes.
				p.Stop()
			}
		}
	}()

	return stop, wait
}

// parseVariableAsHCL parses the value of a single variable as would have been specified
// on the command line via -var or in an environment variable named TF_VAR_x, where x is
// the name of the variable. In order to get around the restriction of HCL requiring a
// top level object, we prepend a sentinel key, decode the user-specified value as its
// value and pull the value back out of the resulting map.
func parseVariableAsHCL(name string, input string, targetType config.VariableType) (interface{}, error) {
	// expecting a string so don't decode anything, just strip quotes
	if targetType == config.VariableTypeString {
		return strings.Trim(input, `"`), nil
	}

	// return empty types
	if strings.TrimSpace(input) == "" {
		switch targetType {
		case config.VariableTypeList:
			return []interface{}{}, nil
		case config.VariableTypeMap:
			return make(map[string]interface{}), nil
		}
	}

	const sentinelValue = "SENTINEL_TERRAFORM_VAR_OVERRIDE_KEY"
	inputWithSentinal := fmt.Sprintf("%s = %s", sentinelValue, input)

	var decoded map[string]interface{}
	err := hcl.Decode(&decoded, inputWithSentinal)
	if err != nil {
		return nil, fmt.Errorf("Cannot parse value for variable %s (%q) as valid HCL: %s", name, input, err)
	}

	if len(decoded) != 1 {
		return nil, fmt.Errorf("Cannot parse value for variable %s (%q) as valid HCL. Only one value may be specified.", name, input)
	}

	parsedValue, ok := decoded[sentinelValue]
	if !ok {
		return nil, fmt.Errorf("Cannot parse value for variable %s (%q) as valid HCL. One value must be specified.", name, input)
	}

	switch targetType {
	case config.VariableTypeList:
		return parsedValue, nil
	case config.VariableTypeMap:
		if list, ok := parsedValue.([]map[string]interface{}); ok {
			return list[0], nil
		}

		return nil, fmt.Errorf("Cannot parse value for variable %s (%q) as valid HCL. One value must be specified.", name, input)
	default:
		panic(fmt.Errorf("unknown type %s", targetType.Printable()))
	}
}
