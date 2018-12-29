package terraform

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/hashicorp/hcl"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/lang"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/provisioners"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
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
	Config    *configs.Config
	Changes   *plans.Changes
	State     *states.State
	Targets   []addrs.Targetable
	Variables InputValues
	Meta      *ContextMeta
	Destroy   bool

	Hooks            []Hook
	Parallelism      int
	ProviderResolver providers.Resolver
	Provisioners     map[string]ProvisionerFactory

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
// NewContext.
type Context struct {
	config    *configs.Config
	changes   *plans.Changes
	state     *states.State
	targets   []addrs.Targetable
	variables InputValues
	meta      *ContextMeta
	destroy   bool

	hooks      []Hook
	components contextComponentFactory
	schemas    *Schemas
	sh         *stopHook
	uiInput    UIInput

	l                   sync.Mutex // Lock acquired during any task
	parallelSem         Semaphore
	providerInputConfig map[string]map[string]cty.Value
	providerSHA256s     map[string][]byte
	runLock             sync.Mutex
	runCond             *sync.Cond
	runContext          context.Context
	runContextCancel    context.CancelFunc
	shadowErr           error
}

// (additional methods on Context can be found in context_*.go files.)

// NewContext creates a new Context structure.
//
// Once a Context is created, the caller must not access or mutate any of
// the objects referenced (directly or indirectly) by the ContextOpts fields.
//
// If the returned diagnostics contains errors then the resulting context is
// invalid and must not be used.
func NewContext(opts *ContextOpts) (*Context, tfdiags.Diagnostics) {
	diags := CheckCoreVersionRequirements(opts.Config)
	// If version constraints are not met then we'll bail early since otherwise
	// we're likely to just see a bunch of other errors related to
	// incompatibilities, which could be overwhelming for the user.
	if diags.HasErrors() {
		return nil, diags
	}

	// Copy all the hooks and add our stop hook. We don't append directly
	// to the Config so that we're not modifying that in-place.
	sh := new(stopHook)
	hooks := make([]Hook, len(opts.Hooks)+1)
	copy(hooks, opts.Hooks)
	hooks[len(opts.Hooks)] = sh

	state := opts.State
	if state == nil {
		state = states.NewState()
	}

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
	var variables InputValues
	if opts.Config != nil {
		// Default variables from the configuration seed our map.
		variables = DefaultVariableValues(opts.Config.Module.Variables)
	}
	// Variables provided by the caller (from CLI, environment, etc) can
	// override the defaults.
	variables = variables.Override(opts.Variables)

	// Bind available provider plugins to the constraints in config
	var providerFactories map[string]providers.Factory
	if opts.ProviderResolver != nil {
		var err error
		deps := ConfigTreeDependencies(opts.Config, state)
		reqd := deps.AllPluginRequirements()
		if opts.ProviderSHA256s != nil && !opts.SkipProviderVerify {
			reqd.LockExecutables(opts.ProviderSHA256s)
		}
		providerFactories, err = resourceProviderFactories(opts.ProviderResolver, reqd)
		if err != nil {
			diags = diags.Append(err)
			return nil, diags
		}
	} else {
		providerFactories = make(map[string]providers.Factory)
	}

	components := &basicComponentFactory{
		providers:    providerFactories,
		provisioners: opts.Provisioners,
	}

	schemas, err := LoadSchemas(opts.Config, opts.State, components)
	if err != nil {
		diags = diags.Append(err)
		return nil, diags
	}

	changes := opts.Changes
	if changes == nil {
		changes = plans.NewChanges()
	}

	config := opts.Config
	if config == nil {
		config = configs.NewEmptyConfig()
	}

	return &Context{
		components: components,
		schemas:    schemas,
		destroy:    opts.Destroy,
		changes:    changes,
		hooks:      hooks,
		meta:       opts.Meta,
		config:     config,
		state:      state,
		targets:    opts.Targets,
		uiInput:    opts.UIInput,
		variables:  variables,

		parallelSem:         NewSemaphore(par),
		providerInputConfig: make(map[string]map[string]cty.Value),
		providerSHA256s:     opts.ProviderSHA256s,
		sh:                  sh,
	}, nil
}

func (c *Context) Schemas() *Schemas {
	return c.schemas
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
func (c *Context) Graph(typ GraphType, opts *ContextGraphOpts) (*Graph, tfdiags.Diagnostics) {
	if opts == nil {
		opts = &ContextGraphOpts{Validate: true}
	}

	log.Printf("[INFO] terraform: building graph: %s", typ)
	switch typ {
	case GraphTypeApply:
		return (&ApplyGraphBuilder{
			Config:     c.config,
			Changes:    c.changes,
			State:      c.state,
			Components: c.components,
			Schemas:    c.schemas,
			Targets:    c.targets,
			Destroy:    c.destroy,
			Validate:   opts.Validate,
		}).Build(addrs.RootModuleInstance)

	case GraphTypeValidate:
		// The validate graph is just a slightly modified plan graph
		fallthrough
	case GraphTypePlan:
		// Create the plan graph builder
		p := &PlanGraphBuilder{
			Config:     c.config,
			State:      c.state,
			Components: c.components,
			Schemas:    c.schemas,
			Targets:    c.targets,
			Validate:   opts.Validate,
		}

		// Some special cases for other graph types shared with plan currently
		var b GraphBuilder = p
		switch typ {
		case GraphTypeValidate:
			b = ValidateGraphBuilder(p)
		}

		return b.Build(addrs.RootModuleInstance)

	case GraphTypePlanDestroy:
		return (&DestroyPlanGraphBuilder{
			Config:     c.config,
			State:      c.state,
			Components: c.components,
			Schemas:    c.schemas,
			Targets:    c.targets,
			Validate:   opts.Validate,
		}).Build(addrs.RootModuleInstance)

	case GraphTypeRefresh:
		return (&RefreshGraphBuilder{
			Config:     c.config,
			State:      c.state,
			Components: c.components,
			Schemas:    c.schemas,
			Targets:    c.targets,
			Validate:   opts.Validate,
		}).Build(addrs.RootModuleInstance)

	case GraphTypeEval:
		return (&EvalGraphBuilder{
			Config:     c.config,
			State:      c.state,
			Components: c.components,
			Schemas:    c.schemas,
		}).Build(addrs.RootModuleInstance)

	default:
		// Should never happen, because the above is exhaustive for all graph types.
		panic(fmt.Errorf("unsupported graph type %s", typ))
	}
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
func (c *Context) State() *states.State {
	return c.state.DeepCopy()
}

// Eval produces a scope in which expressions can be evaluated for
// the given module path.
//
// This method must first evaluate any ephemeral values (input variables, local
// values, and output values) in the configuration. These ephemeral values are
// not included in the persisted state, so they must be re-computed using other
// values in the state before they can be properly evaluated. The updated
// values are retained in the main state associated with the receiving context.
//
// This function takes no action against remote APIs but it does need access
// to all provider and provisioner instances in order to obtain their schemas
// for type checking.
//
// The result is an evaluation scope that can be used to resolve references
// against the root module. If the returned diagnostics contains errors then
// the returned scope may be nil. If it is not nil then it may still be used
// to attempt expression evaluation or other analysis, but some expressions
// may not behave as expected.
func (c *Context) Eval(path addrs.ModuleInstance) (*lang.Scope, tfdiags.Diagnostics) {
	// This is intended for external callers such as the "terraform console"
	// command. Internally, we create an evaluator in c.walk before walking
	// the graph, and create scopes in ContextGraphWalker.

	var diags tfdiags.Diagnostics
	defer c.acquireRun("eval")()

	// Start with a copy of state so that we don't affect any instances
	// that other methods may have already returned.
	c.state = c.state.DeepCopy()
	var walker *ContextGraphWalker

	graph, graphDiags := c.Graph(GraphTypeEval, nil)
	diags = diags.Append(graphDiags)
	if !diags.HasErrors() {
		var walkDiags tfdiags.Diagnostics
		walker, walkDiags = c.walk(graph, walkEval)
		diags = diags.Append(walker.NonFatalDiagnostics)
		diags = diags.Append(walkDiags)
	}

	if walker == nil {
		// If we skipped walking the graph (due to errors) then we'll just
		// use a placeholder graph walker here, which'll refer to the
		// unmodified state.
		walker = c.graphWalker(walkEval)
	}

	// This is a bit weird since we don't normally evaluate outside of
	// the context of a walk, but we'll "re-enter" our desired path here
	// just to get hold of an EvalContext for it. GraphContextBuiltin
	// caches its contexts, so we should get hold of the context that was
	// previously used for evaluation here, unless we skipped walking.
	evalCtx := walker.EnterPath(path)
	return evalCtx.EvaluationScope(nil, EvalDataForNoInstanceKey), diags
}

// Interpolater is no longer used. Use Evaluator instead.
//
// The interpolator returned from this function will return an error on any use.
func (c *Context) Interpolater() *Interpolater {
	// FIXME: Remove this once all callers are updated to no longer use it.
	return &Interpolater{}
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
func (c *Context) Apply() (*states.State, tfdiags.Diagnostics) {
	defer c.acquireRun("apply")()

	// Copy our own state
	c.state = c.state.DeepCopy()

	// Build the graph.
	graph, diags := c.Graph(GraphTypeApply, nil)
	if diags.HasErrors() {
		return nil, diags
	}

	// Determine the operation
	operation := walkApply
	if c.destroy {
		operation = walkDestroy
	}

	// Walk the graph
	walker, walkDiags := c.walk(graph, operation)
	diags = diags.Append(walker.NonFatalDiagnostics)
	diags = diags.Append(walkDiags)

	return c.state, diags
}

// Plan generates an execution plan for the given context.
//
// The execution plan encapsulates the context and can be stored
// in order to reinstantiate a context later for Apply.
//
// Plan also updates the diff of this context to be the diff generated
// by the plan, so Apply can be called after.
func (c *Context) Plan() (*plans.Plan, tfdiags.Diagnostics) {
	defer c.acquireRun("plan")()

	var diags tfdiags.Diagnostics

	varVals := make(map[string]plans.DynamicValue, len(c.variables))
	for k, iv := range c.variables {
		// We use cty.DynamicPseudoType here so that we'll save both the
		// value _and_ its dynamic type in the plan, so we can recover
		// exactly the same value later.
		dv, err := plans.NewDynamicValue(iv.Value, cty.DynamicPseudoType)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to prepare variable value for plan",
				fmt.Sprintf("The value for variable %q could not be serialized to store in the plan: %s.", k, err),
			))
			continue
		}
		varVals[k] = dv
	}

	p := &plans.Plan{
		VariableValues:  varVals,
		TargetAddrs:     c.targets,
		ProviderSHA256s: c.providerSHA256s,
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
			c.state = states.NewState()
		} else {
			c.state = old.DeepCopy()
		}
		defer func() {
			c.state = old
		}()

		operation = walkPlan
	}

	// Build the graph.
	graphType := GraphTypePlan
	if c.destroy {
		graphType = GraphTypePlanDestroy
	}
	graph, graphDiags := c.Graph(graphType, nil)
	diags = diags.Append(graphDiags)
	if graphDiags.HasErrors() {
		return nil, diags
	}

	// Do the walk
	walker, walkDiags := c.walk(graph, operation)
	diags = diags.Append(walker.NonFatalDiagnostics)
	diags = diags.Append(walkDiags)
	if walkDiags.HasErrors() {
		return nil, diags
	}
	p.Changes = c.changes

	return p, diags
}

// Refresh goes through all the resources in the state and refreshes them
// to their latest state. This will update the state that this context
// works with, along with returning it.
//
// Even in the case an error is returned, the state may be returned and
// will potentially be partially updated.
func (c *Context) Refresh() (*states.State, tfdiags.Diagnostics) {
	defer c.acquireRun("refresh")()

	// Copy our own state
	c.state = c.state.DeepCopy()

	// Refresh builds a partial changeset as part of its work because it must
	// create placeholder stubs for any resource instances that'll be created
	// in subsequent plan so that provider configurations and data resources
	// can interpolate from them. This plan is always thrown away after
	// the operation completes, restoring any existing changeset.
	oldChanges := c.changes
	defer func() { c.changes = oldChanges }()
	c.changes = plans.NewChanges()

	// Build the graph.
	graph, diags := c.Graph(GraphTypeRefresh, nil)
	if diags.HasErrors() {
		return nil, diags
	}

	// Do the walk
	_, walkDiags := c.walk(graph, walkRefresh)
	diags = diags.Append(walkDiags)
	if walkDiags.HasErrors() {
		return nil, diags
	}

	// During our walk we will have created planned object placeholders in
	// state for resource instances that are in configuration but not yet
	// created. These were created only to allow expression evaluation to
	// work properly in provider and data blocks during the walk and must
	// now be discarded, since a subsequent plan walk is responsible for
	// creating these "for real".
	// TODO: Consolidate refresh and plan into a single walk, so that the
	// refresh walk doesn't need to emulate various aspects of the plan
	// walk in order to properly evaluate provider and data blocks.
	c.state.SyncWrapper().RemovePlannedResourceInstanceObjects()

	return c.state, diags
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

// Validate performs semantic validation of the configuration, and returning
// any warnings or errors.
//
// Syntax and structural checks are performed by the configuration loader,
// and so are not repeated here.
func (c *Context) Validate() tfdiags.Diagnostics {
	defer c.acquireRun("validate")()

	var diags tfdiags.Diagnostics

	// Validate input variables. We do this only for the values supplied
	// by the root module, since child module calls are validated when we
	// visit their graph nodes.
	if c.config != nil {
		varDiags := checkInputVariables(c.config.Module.Variables, c.variables)
		diags = diags.Append(varDiags)
	}

	// If we have errors at this point then we probably won't be able to
	// construct a graph without producing redundant errors, so we'll halt early.
	if diags.HasErrors() {
		return diags
	}

	// Build the graph so we can walk it and run Validate on nodes.
	// We also validate the graph generated here, but this graph doesn't
	// necessarily match the graph that Plan will generate, so we'll validate the
	// graph again later after Planning.
	graph, graphDiags := c.Graph(GraphTypeValidate, nil)
	diags = diags.Append(graphDiags)
	if graphDiags.HasErrors() {
		return diags
	}

	// Walk
	walker, walkDiags := c.walk(graph, walkValidate)
	diags = diags.Append(walker.NonFatalDiagnostics)
	diags = diags.Append(walkDiags)
	if walkDiags.HasErrors() {
		return diags
	}

	return diags
}

// Config returns the configuration tree associated with this context.
func (c *Context) Config() *configs.Config {
	return c.config
}

// Variables will return the mapping of variables that were defined
// for this Context. If Input was called, this mapping may be different
// than what was given.
func (c *Context) Variables() InputValues {
	return c.variables
}

// SetVariable sets a variable after a context has already been built.
func (c *Context) SetVariable(k string, v cty.Value) {
	c.variables[k] = &InputValue{
		Value:      v,
		SourceType: ValueFromCaller,
	}
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

func (c *Context) walk(graph *Graph, operation walkOperation) (*ContextGraphWalker, tfdiags.Diagnostics) {
	log.Printf("[DEBUG] Starting graph walk: %s", operation.String())

	walker := c.graphWalker(operation)

	// Watch for a stop so we can call the provider Stop() API.
	watchStop, watchWait := c.watchStop(walker)

	// Walk the real graph, this will block until it completes
	diags := graph.Walk(walker)

	// Close the channel so the watcher stops, and wait for it to return.
	close(watchStop)
	<-watchWait

	return walker, diags
}

func (c *Context) graphWalker(operation walkOperation) *ContextGraphWalker {
	return &ContextGraphWalker{
		Context:            c,
		State:              c.state.SyncWrapper(),
		Changes:            c.changes.SyncWrapper(),
		Operation:          operation,
		StopContext:        c.runContext,
		RootVariableValues: c.variables,
	}
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
			ps := make([]providers.Interface, 0, len(walker.providerCache))
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
			ps := make([]provisioners.Interface, 0, len(walker.provisionerCache))
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
