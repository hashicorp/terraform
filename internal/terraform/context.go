package terraform

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/provisioners"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders"
	_ "github.com/hashicorp/terraform/internal/logging"
)

// InputMode defines what sort of input will be asked for when Input
// is called on Context.
type InputMode byte

const (
	// InputModeProvider asks for provider variables
	InputModeProvider InputMode = 1 << iota

	// InputModeStd is the standard operating mode and asks for both variables
	// and providers.
	InputModeStd = InputModeProvider
)

// ContextOpts are the user-configurable options to create a context with
// NewContext.
type ContextOpts struct {
	Config       *configs.Config
	Changes      *plans.Changes
	State        *states.State
	Targets      []addrs.Targetable
	ForceReplace []addrs.AbsResourceInstance
	Variables    InputValues
	Meta         *ContextMeta
	PlanMode     plans.Mode
	SkipRefresh  bool

	Hooks        []Hook
	Parallelism  int
	Providers    map[addrs.Provider]providers.Factory
	Provisioners map[string]provisioners.Factory

	// If non-nil, will apply as additional constraints on the provider
	// plugins that will be requested from the provider resolver.
	ProviderSHA256s map[string][]byte

	// If non-nil, will be verified to ensure that provider requirements from
	// configuration can be satisfied by the set of locked dependencies.
	LockedDependencies *depsfile.Locks

	// Set of providers to exclude from the requirements check process, as they
	// are marked as in local development.
	ProvidersInDevelopment map[addrs.Provider]struct{}

	UIInput UIInput
}

// ContextMeta is metadata about the running context. This is information
// that this package or structure cannot determine on its own but exposes
// into Terraform in various ways. This must be provided by the Context
// initializer.
type ContextMeta struct {
	Env string // Env is the state environment

	// OriginalWorkingDir is the working directory where the Terraform CLI
	// was run from, which may no longer actually be the current working
	// directory if the user included the -chdir=... option.
	//
	// If this string is empty then the original working directory is the same
	// as the current working directory.
	//
	// In most cases we should respect the user's override by ignoring this
	// path and just using the current working directory, but this is here
	// for some exceptional cases where the original working directory is
	// needed.
	OriginalWorkingDir string
}

// Context represents all the context that Terraform needs in order to
// perform operations on infrastructure. This structure is built using
// NewContext.
type Context struct {
	config       *configs.Config
	changes      *plans.Changes
	skipRefresh  bool
	targets      []addrs.Targetable
	forceReplace []addrs.AbsResourceInstance
	variables    InputValues
	meta         *ContextMeta
	planMode     plans.Mode

	// state, refreshState, and prevRunState simultaneously track three
	// different incarnations of the Terraform state:
	//
	// "state" is always the most "up-to-date". During planning it represents
	// our best approximation of the planned new state, and during applying
	// it represents the results of all of the actions we've taken so far.
	//
	// "refreshState" is populated and relevant only during planning, where we
	// update it to reflect a provider's sense of the current state of the
	// remote object each resource instance is bound to but don't include
	// any changes implied by the configuration.
	//
	// "prevRunState" is similar to refreshState except that it doesn't even
	// include the result of the provider's refresh step, and instead reflects
	// the state as we found it prior to any changes, although it does reflect
	// the result of running the provider's schema upgrade actions so that the
	// resource instance objects will all conform to the _current_ resource
	// type schemas if planning is successful, so that in that case it will
	// be meaningful to compare prevRunState to refreshState to detect changes
	// made outside of Terraform.
	state        *states.State
	refreshState *states.State
	prevRunState *states.State

	hooks      []Hook
	components contextComponentFactory
	schemas    *Schemas
	sh         *stopHook
	uiInput    UIInput

	l                   sync.Mutex // Lock acquired during any task
	parallelSem         Semaphore
	providerInputConfig map[string]map[string]cty.Value
	providerSHA256s     map[string][]byte
	runCond             *sync.Cond
	runContext          context.Context
	runContextCancel    context.CancelFunc
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
	log.Printf("[TRACE] terraform.NewContext: starting")
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
	// We throw an error in case of negative parallelism
	par := opts.Parallelism
	if par < 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid parallelism value",
			fmt.Sprintf("The parallelism must be a positive value. Not %d.", par),
		))
		return nil, diags
	}

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

	components := &basicComponentFactory{
		providers:    opts.Providers,
		provisioners: opts.Provisioners,
	}

	log.Printf("[TRACE] terraform.NewContext: loading provider schemas")
	schemas, err := LoadSchemas(opts.Config, opts.State, components)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Could not load plugin",
			fmt.Sprintf(errPluginInit, err),
		))
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

	// TODO: Also apply the moves to the state and changes.
	_, movedDiags := decodeMoves(config, schemas)
	diags = diags.Append(movedDiags)

	// If we have a configuration and a set of locked dependencies, verify that
	// the provider requirements from the configuration can be satisfied by the
	// locked dependencies.
	if opts.LockedDependencies != nil {
		reqs, providerDiags := config.ProviderRequirements()
		diags = diags.Append(providerDiags)

		locked := opts.LockedDependencies.AllProviders()
		unmetReqs := make(getproviders.Requirements)
		for provider, versionConstraints := range reqs {
			// Builtin providers are not listed in the locks file
			if provider.IsBuiltIn() {
				continue
			}
			// Development providers must be excluded from this check
			if _, ok := opts.ProvidersInDevelopment[provider]; ok {
				continue
			}
			// If the required provider doesn't exist in the lock, or the
			// locked version doesn't meet the constraints, mark the
			// requirement unmet
			acceptable := versions.MeetingConstraints(versionConstraints)
			if lock, ok := locked[provider]; !ok || !acceptable.Has(lock.Version()) {
				unmetReqs[provider] = versionConstraints
			}
		}

		if len(unmetReqs) > 0 {
			var buf strings.Builder
			for provider, versionConstraints := range unmetReqs {
				fmt.Fprintf(&buf, "\n- %s", provider)
				if len(versionConstraints) > 0 {
					fmt.Fprintf(&buf, " (%s)", getproviders.VersionConstraintsString(versionConstraints))
				}
			}
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Provider requirements cannot be satisfied by locked dependencies",
				fmt.Sprintf("The following required providers are not installed:\n%s\n\nPlease run \"terraform init\".", buf.String()),
			))
			return nil, diags
		}
	}

	switch opts.PlanMode {
	case plans.NormalMode, plans.DestroyMode:
		// OK
	case plans.RefreshOnlyMode:
		if opts.SkipRefresh {
			// The CLI layer (and other similar callers) should prevent this
			// combination of options.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Incompatible plan options",
				"Cannot skip refreshing in refresh-only mode. This is a bug in Terraform.",
			))
			return nil, diags
		}
	default:
		// The CLI layer (and other similar callers) should not try to
		// create a context for a mode that Terraform Core doesn't support.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unsupported plan mode",
			fmt.Sprintf("Terraform Core doesn't know how to handle plan mode %s. This is a bug in Terraform.", opts.PlanMode),
		))
		return nil, diags
	}
	if len(opts.ForceReplace) > 0 && opts.PlanMode != plans.NormalMode {
		// The other modes don't generate no-op or update actions that we might
		// upgrade to be "replace", so doesn't make sense to combine those.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unsupported plan mode",
			fmt.Sprintf("Forcing resource instance replacement (with -replace=...) is allowed only in normal planning mode."),
		))
		return nil, diags
	}

	log.Printf("[TRACE] terraform.NewContext: complete")

	// By the time we get here, we should have values defined for all of
	// the root module variables, even if some of them are "unknown". It's the
	// caller's responsibility to have already handled the decoding of these
	// from the various ways the CLI allows them to be set and to produce
	// user-friendly error messages if they are not all present, and so
	// the error message from checkInputVariables should never be seen and
	// includes language asking the user to report a bug.
	if config != nil {
		varDiags := checkInputVariables(config.Module.Variables, variables)
		diags = diags.Append(varDiags)
	}

	return &Context{
		components:   components,
		schemas:      schemas,
		planMode:     opts.PlanMode,
		changes:      changes,
		hooks:        hooks,
		meta:         opts.Meta,
		config:       config,
		state:        state,
		refreshState: state.DeepCopy(),
		prevRunState: state.DeepCopy(),
		skipRefresh:  opts.SkipRefresh,
		targets:      opts.Targets,
		forceReplace: opts.ForceReplace,
		uiInput:      opts.UIInput,
		variables:    variables,

		parallelSem:         NewSemaphore(par),
		providerInputConfig: make(map[string]map[string]cty.Value),
		providerSHA256s:     opts.ProviderSHA256s,
		sh:                  sh,
	}, diags
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
			Config:       c.config,
			Changes:      c.changes,
			State:        c.state,
			Components:   c.components,
			Schemas:      c.schemas,
			Targets:      c.targets,
			ForceReplace: c.forceReplace,
			Validate:     opts.Validate,
		}).Build(addrs.RootModuleInstance)

	case GraphTypeValidate:
		// The validate graph is just a slightly modified plan graph: an empty
		// state is substituted in for Validate.
		return ValidateGraphBuilder(&PlanGraphBuilder{
			Config:     c.config,
			Components: c.components,
			Schemas:    c.schemas,
			Targets:    c.targets,
			Validate:   opts.Validate,
			State:      states.NewState(),
		}).Build(addrs.RootModuleInstance)

	case GraphTypePlan:
		// Create the plan graph builder
		return (&PlanGraphBuilder{
			Config:       c.config,
			State:        c.state,
			Components:   c.components,
			Schemas:      c.schemas,
			Targets:      c.targets,
			ForceReplace: c.forceReplace,
			Validate:     opts.Validate,
			skipRefresh:  c.skipRefresh,
		}).Build(addrs.RootModuleInstance)

	case GraphTypePlanDestroy:
		return (&DestroyPlanGraphBuilder{
			Config:      c.config,
			State:       c.state,
			Components:  c.components,
			Schemas:     c.schemas,
			Targets:     c.targets,
			Validate:    opts.Validate,
			skipRefresh: c.skipRefresh,
		}).Build(addrs.RootModuleInstance)

	case GraphTypePlanRefreshOnly:
		// Create the plan graph builder, with skipPlanChanges set to
		// activate the "refresh only" mode.
		return (&PlanGraphBuilder{
			Config:          c.config,
			State:           c.state,
			Components:      c.components,
			Schemas:         c.schemas,
			Targets:         c.targets,
			Validate:        opts.Validate,
			skipRefresh:     c.skipRefresh,
			skipPlanChanges: true, // this activates "refresh only" mode.
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
	if c.planMode == plans.DestroyMode {
		operation = walkDestroy
	}

	// Walk the graph
	walker, walkDiags := c.walk(graph, operation)
	diags = diags.Append(walker.NonFatalDiagnostics)
	diags = diags.Append(walkDiags)

	if c.planMode == plans.DestroyMode && !diags.HasErrors() {
		// If we know we were trying to destroy objects anyway, and we
		// completed without any errors, then we'll also prune out any
		// leftover empty resource husks (left after all of the instances
		// of a resource with "count" or "for_each" are destroyed) to
		// help ensure we end up with an _actually_ empty state, assuming
		// we weren't destroying with -target here.
		//
		// (This doesn't actually take into account -target, but that should
		// be okay because it doesn't throw away anything we can't recompute
		// on a subsequent "terraform plan" run, if the resources are still
		// present in the configuration. However, this _will_ cause "count = 0"
		// resources to read as unknown during the next refresh walk, which
		// may cause some additional churn if used in a data resource or
		// provider block, until we remove refreshing as a separate walk and
		// just do it as part of the plan walk.)
		c.state.PruneResourceHusks()
	}

	if len(c.targets) > 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			"Applied changes may be incomplete",
			`The plan was created with the -target option in effect, so some changes requested in the configuration may have been ignored and the output values may not be fully updated. Run the following command to verify that no other changes are pending:
    terraform plan
	
Note that the -target option is not suitable for routine use, and is provided only for exceptional situations such as recovering from errors or mistakes, or when Terraform specifically suggests to use it as part of an error message.`,
		))
	}

	// This isn't technically needed, but don't leave an old refreshed state
	// around in case we re-use the context in internal tests.
	c.refreshState = c.state.DeepCopy()

	return c.state, diags
}

// Plan generates an execution plan for the given context, and returns the
// refreshed state.
//
// The execution plan encapsulates the context and can be stored
// in order to reinstantiate a context later for Apply.
//
// Plan also updates the diff of this context to be the diff generated
// by the plan, so Apply can be called after.
func (c *Context) Plan() (*plans.Plan, tfdiags.Diagnostics) {
	defer c.acquireRun("plan")()
	c.changes = plans.NewChanges()
	var diags tfdiags.Diagnostics

	if len(c.targets) > 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			"Resource targeting is in effect",
			`You are creating a plan with the -target option, which means that the result of this plan may not represent all of the changes requested by the current configuration.
		
The -target option is not for routine use, and is provided only for exceptional situations such as recovering from errors or mistakes, or when Terraform specifically suggests to use it as part of an error message.`,
		))
	}

	var plan *plans.Plan
	var planDiags tfdiags.Diagnostics
	switch c.planMode {
	case plans.NormalMode:
		plan, planDiags = c.plan()
	case plans.DestroyMode:
		plan, planDiags = c.destroyPlan()
	case plans.RefreshOnlyMode:
		plan, planDiags = c.refreshOnlyPlan()
	default:
		panic(fmt.Sprintf("unsupported plan mode %s", c.planMode))
	}
	diags = diags.Append(planDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	// convert the variables into the format expected for the plan
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

	// insert the run-specific data from the context into the plan; variables,
	// targets and provider SHAs.
	plan.VariableValues = varVals
	plan.TargetAddrs = c.targets
	plan.ProviderSHA256s = c.providerSHA256s

	return plan, diags
}

func (c *Context) plan() (*plans.Plan, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	graph, graphDiags := c.Graph(GraphTypePlan, nil)
	diags = diags.Append(graphDiags)
	if graphDiags.HasErrors() {
		return nil, diags
	}

	// Do the walk
	walker, walkDiags := c.walk(graph, walkPlan)
	diags = diags.Append(walker.NonFatalDiagnostics)
	diags = diags.Append(walkDiags)
	if walkDiags.HasErrors() {
		return nil, diags
	}
	plan := &plans.Plan{
		UIMode:            plans.NormalMode,
		Changes:           c.changes,
		ForceReplaceAddrs: c.forceReplace,
		PrevRunState:      c.prevRunState.DeepCopy(),
	}

	c.refreshState.SyncWrapper().RemovePlannedResourceInstanceObjects()

	refreshedState := c.refreshState.DeepCopy()
	plan.PriorState = refreshedState

	// replace the working state with the updated state, so that immediate calls
	// to Apply work as expected.
	c.state = refreshedState

	return plan, diags
}

func (c *Context) destroyPlan() (*plans.Plan, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	destroyPlan := &plans.Plan{
		PriorState: c.state.DeepCopy(),
	}
	c.changes = plans.NewChanges()

	// A destroy plan starts by running Refresh to read any pending data
	// sources, and remove missing managed resources. This is required because
	// a "destroy plan" is only creating delete changes, and is essentially a
	// local operation.
	//
	// NOTE: if skipRefresh _is_ set then we'll rely on the destroy-plan walk
	// below to upgrade the prevRunState and priorState both to the latest
	// resource type schemas, so NodePlanDestroyableResourceInstance.Execute
	// must coordinate with this by taking that action only when c.skipRefresh
	// _is_ set. This coupling between the two is unfortunate but necessary
	// to work within our current structure.
	if !c.skipRefresh {
		refreshPlan, refreshDiags := c.plan()
		diags = diags.Append(refreshDiags)
		if diags.HasErrors() {
			return nil, diags
		}

		// insert the refreshed state into the destroy plan result, and discard
		// the changes recorded from the refresh.
		destroyPlan.PriorState = refreshPlan.PriorState.DeepCopy()
		destroyPlan.PrevRunState = refreshPlan.PrevRunState.DeepCopy()
		c.changes = plans.NewChanges()
	}

	graph, graphDiags := c.Graph(GraphTypePlanDestroy, nil)
	diags = diags.Append(graphDiags)
	if graphDiags.HasErrors() {
		return nil, diags
	}

	// Do the walk
	walker, walkDiags := c.walk(graph, walkPlanDestroy)
	diags = diags.Append(walker.NonFatalDiagnostics)
	diags = diags.Append(walkDiags)
	if walkDiags.HasErrors() {
		return nil, diags
	}

	if c.skipRefresh {
		// If we didn't do refreshing then both the previous run state and
		// the prior state are the result of upgrading the previous run state,
		// which we should've upgraded as part of the plan-destroy walk
		// in NodePlanDestroyableResourceInstance.Execute, so they'll have the
		// current schema but neither will reflect any out-of-band changes in
		// the remote system.
		destroyPlan.PrevRunState = c.prevRunState.DeepCopy()
		destroyPlan.PriorState = c.prevRunState.DeepCopy()
	}

	destroyPlan.UIMode = plans.DestroyMode
	destroyPlan.Changes = c.changes
	return destroyPlan, diags
}

func (c *Context) refreshOnlyPlan() (*plans.Plan, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	graph, graphDiags := c.Graph(GraphTypePlanRefreshOnly, nil)
	diags = diags.Append(graphDiags)
	if graphDiags.HasErrors() {
		return nil, diags
	}

	// Do the walk
	walker, walkDiags := c.walk(graph, walkPlan)
	diags = diags.Append(walker.NonFatalDiagnostics)
	diags = diags.Append(walkDiags)
	if walkDiags.HasErrors() {
		return nil, diags
	}
	plan := &plans.Plan{
		UIMode:       plans.RefreshOnlyMode,
		Changes:      c.changes,
		PrevRunState: c.prevRunState.DeepCopy(),
	}

	// If the graph builder and graph nodes correctly obeyed our directive
	// to refresh only, the set of resource changes should always be empty.
	// We'll safety-check that here so we can return a clear message about it,
	// rather than probably just generating confusing output at the UI layer.
	if len(plan.Changes.Resources) != 0 {
		// Some extra context in the logs in case the user reports this message
		// as a bug, as a starting point for debugging.
		for _, rc := range plan.Changes.Resources {
			if depKey := rc.DeposedKey; depKey == states.NotDeposed {
				log.Printf("[DEBUG] Refresh-only plan includes %s change for %s", rc.Action, rc.Addr)
			} else {
				log.Printf("[DEBUG] Refresh-only plan includes %s change for %s deposed object %s", rc.Action, rc.Addr, depKey)
			}
		}
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid refresh-only plan",
			"Terraform generated planned resource changes in a refresh-only plan. This is a bug in Terraform.",
		))
	}

	c.refreshState.SyncWrapper().RemovePlannedResourceInstanceObjects()

	refreshedState := c.refreshState
	plan.PriorState = refreshedState.DeepCopy()

	// replace the working state with the updated state, so that immediate calls
	// to Apply work as expected. DeepCopy because such an apply should not
	// mutate
	c.state = refreshedState

	return plan, diags
}

// Refresh goes through all the resources in the state and refreshes them
// to their latest state. This is done by executing a plan, and retaining the
// state while discarding the change set.
//
// In the case of an error, there is no state returned.
func (c *Context) Refresh() (*states.State, tfdiags.Diagnostics) {
	p, diags := c.Plan()
	if diags.HasErrors() {
		return nil, diags
	}

	return p.PriorState, diags
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
		log.Printf("[INFO] terraform: waiting for graceful stop to complete")
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
	var state *states.SyncState
	var refreshState *states.SyncState
	var prevRunState *states.SyncState

	switch operation {
	case walkValidate:
		// validate should not use any state
		state = states.NewState().SyncWrapper()

		// validate currently uses the plan graph, so we have to populate the
		// refreshState and the prevRunState.
		refreshState = states.NewState().SyncWrapper()
		prevRunState = states.NewState().SyncWrapper()

	case walkPlan, walkPlanDestroy:
		state = c.state.SyncWrapper()
		refreshState = c.refreshState.SyncWrapper()
		prevRunState = c.prevRunState.SyncWrapper()

	default:
		state = c.state.SyncWrapper()
	}

	return &ContextGraphWalker{
		Context:            c,
		State:              state,
		RefreshState:       refreshState,
		PrevRunState:       prevRunState,
		Changes:            c.changes.SyncWrapper(),
		InstanceExpander:   instances.NewExpander(),
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
		log.Printf("[TRACE] Context: requesting providers and provisioners to gracefully stop")

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
