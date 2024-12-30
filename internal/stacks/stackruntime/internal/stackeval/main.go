// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hashicorp/go-slug/sourcebundle"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"

	"github.com/hashicorp/terraform/internal/addrs"
	fileProvisioner "github.com/hashicorp/terraform/internal/builtin/provisioners/file"
	remoteExecProvisioner "github.com/hashicorp/terraform/internal/builtin/provisioners/remote-exec"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/provisioners"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/internal/stackeval/stubs"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Main is the central node of all data required for performing the major
// actions against a stack: validation, planning, and applying.
//
// This type delegates to various other types in this package to implement
// the real logic, with Main focused on enabling the collaboration between
// objects of those other types.
type Main struct {
	config *stackconfig.Config

	// validating captures the data needed when validating configuration,
	// in which case we consider _only_ the configuration and don't take
	// into account any existing state or specific input variable values.
	validating *mainValidating

	// planning captures the data needed when creating or applying a plan,
	// but which need not be populated when only using the validation-related
	// functionality of this package.
	planning *mainPlanning

	// applying captures the data needed when applying a plan. This can
	// only be populated if "planning" is also populated, because the process
	// of applying includes re-evaluation of plans based on new information
	// gathered during the apply process.
	applying *mainApplying

	// inspecting captures the data needed when operating in the special
	// "inspect" mode, which doesn't plan or apply but can still evaluate
	// expressions and inspect the current configuration/state.
	inspecting *mainInspecting

	// providerFactories is a set of callback functions through which the
	// runtime can obtain new instances of each of the available providers.
	providerFactories ProviderFactories

	// testOnlyGlobals is a unit testing affordance: if non-nil, expressions
	// shaped like _test_only_global.name (with the leading underscore)
	// become valid throughout the stack configuration and evaluate to the
	// correspondingly-named values here.
	//
	// This must never be used outside of test code in this package.
	testOnlyGlobals map[string]cty.Value

	// languageExperimentsAllowed gets set if our caller enables the use
	// of language experiments by calling [Main.AllowLanguageExperiments]
	// shortly after creating this object.
	languageExperimentsAllowed bool

	// The remaining fields memoize other objects we might create in response
	// to method calls. Must lock "mu" before interacting with them.
	mu                      sync.Mutex
	mainStackConfig         *StackConfig
	mainStack               *Stack
	providerTypes           map[addrs.Provider]*ProviderType
	providerFunctionResults *providers.FunctionResults
	cleanupFuncs            []func(context.Context) tfdiags.Diagnostics
}

var _ namedPromiseReporter = (*Main)(nil)

type mainValidating struct {
	opts ValidateOpts
}

type mainPlanning struct {
	opts      PlanOpts
	prevState *stackstate.State
}

type mainApplying struct {
	opts    ApplyOpts
	plan    *stackplan.Plan
	results *ChangeExecResults
}

type mainInspecting struct {
	opts  InspectOpts
	state *stackstate.State
}

func NewForValidating(config *stackconfig.Config, opts ValidateOpts) *Main {
	return &Main{
		config: config,
		validating: &mainValidating{
			opts: opts,
		},
		providerFactories:       opts.ProviderFactories,
		providerTypes:           make(map[addrs.Provider]*ProviderType),
		providerFunctionResults: providers.NewFunctionResultsTable(nil),
	}
}

func NewForPlanning(config *stackconfig.Config, prevState *stackstate.State, opts PlanOpts) *Main {
	if prevState == nil {
		// We'll make an empty state just to avoid other code needing to deal
		// with this possibly being nil.
		prevState = stackstate.NewState()
	}
	return &Main{
		config: config,
		planning: &mainPlanning{
			opts:      opts,
			prevState: prevState,
		},
		providerFactories:       opts.ProviderFactories,
		providerTypes:           make(map[addrs.Provider]*ProviderType),
		providerFunctionResults: providers.NewFunctionResultsTable(nil),
	}
}

func NewForApplying(config *stackconfig.Config, plan *stackplan.Plan, execResults *ChangeExecResults, opts ApplyOpts) *Main {
	return &Main{
		config: config,
		applying: &mainApplying{
			opts:    opts,
			plan:    plan,
			results: execResults,
		},
		providerFactories:       opts.ProviderFactories,
		providerTypes:           make(map[addrs.Provider]*ProviderType),
		providerFunctionResults: providers.NewFunctionResultsTable(plan.ProviderFunctionResults),
	}
}

func NewForInspecting(config *stackconfig.Config, state *stackstate.State, opts InspectOpts) *Main {
	return &Main{
		config: config,
		inspecting: &mainInspecting{
			state: state,
			opts:  opts,
		},
		providerFactories:       opts.ProviderFactories,
		providerTypes:           make(map[addrs.Provider]*ProviderType),
		providerFunctionResults: providers.NewFunctionResultsTable(nil),
		testOnlyGlobals:         opts.TestOnlyGlobals,
	}
}

// AllowLanguageExperiments changes the flag for whether language experiments
// are allowed during evaluation.
//
// Call this very shortly after creating a [Main], before performing any other
// actions on it. Changing this setting after other methods have been called
// will produce unpredictable results.
func (m *Main) AllowLanguageExperiments(allow bool) {
	m.languageExperimentsAllowed = allow
}

// LanguageExperimentsAllowed returns true if language experiments are allowed
// to be used during evaluation.
func (m *Main) LanguageExperimentsAllowed() bool {
	return m.languageExperimentsAllowed
}

// Validating returns true if the receiving [Main] is configured for validating.
//
// If this returns false then validation methods may panic or return strange
// results.
func (m *Main) Validating() bool {
	return m.validating != nil
}

// Planning returns true if the receiving [Main] is configured for planning.
//
// If this returns false then planning methods may panic or return strange
// results.
func (m *Main) Planning() bool {
	return m.planning != nil
}

// Applying returns true if the receiving [Main] is configured for applying.
//
// If this returns false then applying methods may panic or return strange
// results.
func (m *Main) Applying() bool {
	return m.applying != nil
}

// Inspecting returns true if the receiving [Main] is configured for inspecting.
//
// If this returns false then expression evaluation in [InspectPhase] is
// likely to panic or return strange results.
func (m *Main) Inspecting() bool {
	return m.inspecting != nil
}

// PlanningOpts returns the planning options to use during the planning phase,
// or panics if this [Main] was not instantiated for planning.
//
// Do not modify anything reachable through the returned pointer.
func (m *Main) PlanningOpts() *PlanOpts {
	if !m.Planning() {
		panic("stacks language runtime is not instantiated for planning")
	}
	return &m.planning.opts
}

// PlanPrevState returns the "previous state" object to use as the basis for
// planning, or panics if this [Main] is not instantiated for planning.
func (m *Main) PlanPrevState() *stackstate.State {
	if !m.Planning() {
		panic("previous state is only available in the planning phase")
	}
	return m.planning.prevState
}

// ApplyChangeResults returns the object that tracks the results of the actual
// changes being made during the apply phase, or panics if this [Main] is not
// instantiated for applying.
func (m *Main) ApplyChangeResults() *ChangeExecResults {
	if !m.Applying() {
		panic("stacks language runtime is not instantiated for applying")
	}
	if m.applying.results == nil {
		panic("stacks language runtime is instantiated for applying but somehow has no change results")
	}
	return m.applying.results
}

// PlanBeingApplied returns the plan that's currently being applied, or panics
// if called not during an apply phase.
func (m *Main) PlanBeingApplied() *stackplan.Plan {
	if !m.Applying() {
		panic("stacks language runtime is not instantiated for applying")
	}
	return m.applying.plan
}

// InspectingState returns the state snapshot that was provided when
// instantiating [Main] "for inspecting", or panics if this object was not
// instantiated in that mode.
func (m *Main) InspectingState() *stackstate.State {
	if !m.Inspecting() {
		panic("stacks language runtime is not instantiated for inspecting")
	}
	return m.inspecting.state
}

// SourceBundle returns the source code bundle that the stack configuration
// was originally loaded from and that should also contain the source code
// for any modules that "component" blocks refer to.
func (m *Main) SourceBundle(ctx context.Context) *sourcebundle.Bundle {
	return m.config.Sources
}

// MainStackConfig returns the [StackConfig] object representing the main
// stack configuration, which is at the root of the configuration tree.
//
// This represents the static configuration. The main stack configuration
// always has exactly one "dynamic" instance, which you can access by
// calling [Main.MainStack] instead. The static configuration object is used
// for validation, but plan and apply both use the stack instance.
func (m *Main) MainStackConfig(ctx context.Context) *StackConfig {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.mainStackConfig == nil {
		m.mainStackConfig = newStackConfig(m, stackaddrs.RootStack, m.config.Root)
	}
	return m.mainStackConfig
}

// MainStack returns the [Stack] object representing the main stack, which
// is the root of the configuration tree.
func (m *Main) MainStack(ctx context.Context) *Stack {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.mainStack == nil {
		m.mainStack = newStack(m, stackaddrs.RootStackInstance)
	}
	return m.mainStack
}

// StackConfig returns the [StackConfig] object representing the stack with
// the given address, or nil if there is no such stack.
func (m *Main) StackConfig(ctx context.Context, addr stackaddrs.Stack) *StackConfig {
	ret := m.MainStackConfig(ctx)
	for _, step := range addr {
		ret = ret.ChildConfig(ctx, step)
		if ret == nil {
			return nil
		}
	}
	return ret
}

// StackUnchecked returns the [Stack] object representing the stack instance
// with the given address, or nil if the address traverses through an embedded
// stack call that doesn't exist at all.
//
// This function cannot check whether the instance keys in the path correspond
// to instances actually declared by the configuration. If you need to check
// that use [Main.Stack] instead, but consider the additional overhead that
// extra checking implies.
func (m *Main) StackUnchecked(ctx context.Context, addr stackaddrs.StackInstance) *Stack {
	ret := m.MainStack(ctx)
	for _, step := range addr {
		ret = ret.ChildStackUnchecked(ctx, step)
		if ret == nil {
			return nil
		}
	}
	return ret
}

// Stack returns the [Stack] object representing the stack instance with the
// given address, or nil if we know for certain that such a stack instance
// is not declared in the configuration.
//
// This is like [Main.StackUnchecked] but additionally checks whether all of
// the instance keys in the stack instance path correspond to instances declared
// by for_each arguments (or lack thereof) in the configuration. This involves
// evaluating all of the "for_each" expressions, and so will block on whatever
// those expressions depend on.
//
// If any of the stack calls along the path have an as-yet-unknown set of
// instances, this function will optimistically return a non-nil stack but
// further operations with that stack are likely to return unknown values
// themselves.
//
// If you know you are holding a [stackaddrs.StackInstance] that was from
// a valid [Stack] previously returned (directly or indirectly) then you can
// avoid the additional overhead by using [Main.StackUnchecked] instead.
func (m *Main) Stack(ctx context.Context, addr stackaddrs.StackInstance, phase EvalPhase) *Stack {
	ret := m.MainStack(ctx)
	for _, step := range addr {
		ret = ret.ChildStackChecked(ctx, step, phase)
		if ret == nil {
			return nil
		}
	}
	return ret
}

// ProviderFactories returns the collection of factory functions for providers
// that are available to this instance of the evaluation runtime.
func (m *Main) ProviderFactories() ProviderFactories {
	return m.providerFactories
}

// ProviderFunctions returns the collection of externally defined provider
// functions available to the current stack.
func (m *Main) ProviderFunctions(ctx context.Context, config *StackConfig) (lang.ExternalFuncs, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	fns := make(map[string]map[string]function.Function, len(m.providerFactories))

	for addr := range m.providerFactories {
		provider := m.ProviderType(ctx, addr)

		local, ok := config.ProviderLocalName(ctx, addr)
		if !ok {
			log.Printf("[ERROR] Provider %s is not in the required providers block", addr)
			// This also shouldn't happen, as every provider should be
			// in the required providers block and that should have been
			// validated - but we can recover from this by just using the
			// default local name.
			local = addr.Type
		}

		schema, err := provider.Schema(ctx)
		if err != nil {
			// We should have started these providers before we got here, so
			// this error shouldn't ever occur.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Failed to retrieve provider schema",
				Detail:   fmt.Sprintf("Failed to retrieve schema for provider %s while gathering provider functions: %s. This is a bug in Terraform, please report it!", addr, err),
			})
			continue // just skip this provider and keep going
		}

		// Now we can build the functions for this provider.
		fns[local] = make(map[string]function.Function, len(schema.Functions))
		for name, fn := range schema.Functions {
			fns[local][name] = fn.BuildFunction(addr, name, m.providerFunctionResults, func() (providers.Interface, error) {
				client, err := provider.UnconfiguredClient()
				if err != nil {
					return nil, err
				}
				return stubs.OfflineProvider(client), nil
			})
		}
	}

	return lang.ExternalFuncs{Provider: fns}, diags

}

// ProviderType returns the [ProviderType] object representing the given
// provider source address.
//
// This does not check whether the given provider type is available in the
// current evaluation context, but attempting to create a client for a
// provider that isn't available will return an error at startup time.
func (m *Main) ProviderType(ctx context.Context, addr addrs.Provider) *ProviderType {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.providerTypes[addr] == nil {
		m.providerTypes[addr] = newProviderType(m, addr)
	}
	return m.providerTypes[addr]
}

func (m *Main) ProviderRefTypes() map[addrs.Provider]cty.Type {
	return m.config.ProviderRefTypes
}

// PreviousProviderInstances fetches the set of providers that are required
// based on the current plan or state file. They are previous in the sense that
// they're not based on the current config. So if a provider has been removed
// from the config, this function will still find it.
func (m *Main) PreviousProviderInstances(addr stackaddrs.AbsComponentInstance, phase EvalPhase) addrs.Set[addrs.RootProviderConfig] {
	switch phase {
	case ApplyPhase:
		return m.PlanBeingApplied().RequiredProviderInstances(addr)
	case PlanPhase:
		return m.PlanPrevState().RequiredProviderInstances(addr)
	case InspectPhase:
		return m.InspectingState().RequiredProviderInstances(addr)
	default:
		// We don't have the required information (like a plan or a state file)
		// in the other phases so we can't do anything even if we wanted to.
		// In general, for the other phases we're not doing anything with the
		// previous provider instances anyway, so we don't need them.
		return addrs.MakeSet[addrs.RootProviderConfig]()
	}
}

// RootVariableValue returns the original root variable value specified by the
// caller, if any. The caller of this function is responsible for replacing
// missing values with defaults, and performing type conversion and and
// validation.
func (m *Main) RootVariableValue(ctx context.Context, addr stackaddrs.InputVariable, phase EvalPhase) ExternalInputValue {
	switch phase {
	case PlanPhase:
		if !m.Planning() {
			panic("using PlanPhase input variable values when not configured for planning")
		}
		ret, ok := m.planning.opts.InputVariableValues[addr]
		if !ok {
			// If no value is specified for the given input variable, we return
			// a null value. Callers should treat a null value as equivalent to
			// an unspecified one, applying default (if present) or raising an
			// error (if not).
			return ExternalInputValue{
				Value: cty.NullVal(cty.DynamicPseudoType),
			}
		}
		return ret

	case ApplyPhase:
		if !m.Applying() {
			panic("using ApplyPhase input variable values when not configured for applying")
		}

		// First, check the values given to use directly by the caller.
		if ret, ok := m.applying.opts.InputVariableValues[addr]; ok {
			return ret
		}

		// If the caller didn't provide a value, we need to look up the value
		// that was used during planning.

		if ret, ok := m.applying.plan.RootInputValues[addr]; ok {
			return ExternalInputValue{
				Value: ret,
			}
		}

		// If we had nothing set, we'll return a null value. This means the
		// default value will be applied, if any, or an error will be raised
		// if no default is available. This should only be possible for an
		// ephemeral value in which the caller didn't provide a value during
		// the apply operation.
		return ExternalInputValue{
			Value: cty.NullVal(cty.DynamicPseudoType),
		}

	case InspectPhase:
		if !m.Inspecting() {
			panic("using InspectPhase input variable values when not configured for inspecting")
		}
		ret, ok := m.inspecting.opts.InputVariableValues[addr]
		if !ok {
			return ExternalInputValue{
				// We use the generic "unknown value of unknown type"
				// placeholder here because this method provides the external
				// view of the input variables, and so we expect internal
				// access points like methods of [InputVariable] to convert
				// this result into the appropriate type constraint themselves.
				Value: cty.DynamicVal,
			}
		}
		return ret

	default:
		// Root input variable values are not available in any other phase.

		return ExternalInputValue{
			Value: cty.DynamicVal, // placeholder value
		}
	}
}

// ResolveAbsExpressionReference tries to resolve the given absolute
// expression reference within this evaluation context.
func (m *Main) ResolveAbsExpressionReference(ctx context.Context, ref stackaddrs.AbsReference, phase EvalPhase) (Referenceable, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	stack := m.Stack(ctx, ref.Stack, phase)
	if stack == nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reference to undeclared stack",
			Detail:   fmt.Sprintf("Cannot resolve reference to object in undeclared stack %s.", ref.Stack),
			Subject:  ref.SourceRange().ToHCL().Ptr(),
		})
		return nil, diags
	}
	return stack.ResolveExpressionReference(ctx, ref.Ref)
}

// RegisterCleanup registers an arbitrary callback function to run when a
// walk driver eventually calls [Main.RunCleanup] on the same receiver.
//
// This is intended for cleaning up any resources that would not naturally
// be cleaned up as a result of garbage-collecting the [Main] object and its
// many descendants.
//
// The context passed to a callback function may be already cancelled by the
// time the callback is running, if the cleanup is running in response to
// cancellation.
func (m *Main) RegisterCleanup(cb func(ctx context.Context) tfdiags.Diagnostics) {
	m.mu.Lock()
	m.cleanupFuncs = append(m.cleanupFuncs, cb)
	m.mu.Unlock()
}

// DoCleanup executes any cleanup functions previously registered using
// [Main.RegisterCleanup], returning any collected diagnostics.
//
// Call this only once evaluation has completed and there aren't any requests
// outstanding that might be using resources that this will free. After calling
// this, the [Main] and all other objects created through it become invalid
// and must not be used anymore.
func (m *Main) DoCleanup(ctx context.Context) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	m.mu.Lock()
	funcs := m.cleanupFuncs
	m.cleanupFuncs = nil
	m.mu.Unlock()
	for _, cb := range funcs {
		diags = diags.Append(
			cb(ctx),
		)
	}
	return diags
}

// mustStackConfig is like [Main.StackConfig] except that it panics if it
// does not find a stack configuration object matching the given address,
// for situations where the absense of a stack config represents a bug
// somewhere in Terraform, rather than incorrect user input.
func (m *Main) mustStackConfig(ctx context.Context, addr stackaddrs.Stack) *StackConfig {
	ret := m.StackConfig(ctx, addr)
	if ret == nil {
		panic(fmt.Sprintf("no configuration for %s", addr))
	}
	return ret
}

// StackCallConfig returns the [StackCallConfig] object representing the
// "stack" block in the configuration with the given address, or nil if there
// is no such block.
func (m *Main) StackCallConfig(ctx context.Context, addr stackaddrs.ConfigStackCall) *StackCallConfig {
	caller := m.StackConfig(ctx, addr.Stack)
	if caller == nil {
		return nil
	}
	return caller.StackCall(ctx, addr.Item)
}

// reportNamedPromises implements namedPromiseReporter.
func (m *Main) reportNamedPromises(cb func(id promising.PromiseID, name string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.mainStackConfig != nil {
		m.mainStackConfig.reportNamedPromises(cb)
	}
	if m.mainStack != nil {
		m.mainStack.reportNamedPromises(cb)
	}
	for _, pty := range m.providerTypes {
		pty.reportNamedPromises(cb)
	}
}

// availableProvisioners returns the table of provisioner factories that should
// be made available to modules in this component.
func (m *Main) availableProvisioners() map[string]provisioners.Factory {
	return map[string]provisioners.Factory{
		"remote-exec": func() (provisioners.Interface, error) {
			return remoteExecProvisioner.New(), nil
		},
		"file": func() (provisioners.Interface, error) {
			return fileProvisioner.New(), nil
		},
		"local-exec": func() (provisioners.Interface, error) {
			// We don't yet have any way to ensure a consistent execution
			// environment for local-exec, which means that use of this
			// provisioner is very likely to hurt portability between
			// local and remote usage of stacks. Existing use of local-exec
			// also tends to assume a writable module directory, whereas
			// stack components execute from a read-only directory.
			//
			// Therefore we'll leave this unavailable for now with an explicit
			// error message, although we might revisit this later if there's
			// a strong reason to allow it and if we can find a suitable
			// way to avoid the portability pitfalls that might inhibit
			// moving execution of a stack from one execution environment to
			// another.
			return nil, fmt.Errorf("local-exec provisioners are not supported in stack components; use provider functionality or remote provisioners instead")
		},
	}
}

// PlanTimestamp provides the timestamp at which the plan
// associated with this operation is being executed.
// If we are planning we either take the forced timestamp or the saved current time
// If we are applying we take the timestamp time from the plan
func (m *Main) PlanTimestamp() time.Time {
	if m.applying != nil {
		return m.applying.plan.PlanTimestamp
	}
	if m.planning != nil {
		return m.planning.opts.PlanTimestamp
	}

	// This is the default case, we are not planning / applying
	return time.Now().UTC()
}

// DependencyLocks returns the dependency locks for the given phase.
func (m *Main) DependencyLocks(phase EvalPhase) *depsfile.Locks {
	switch phase {
	case ValidatePhase:
		return &m.validating.opts.DependencyLocks
	case PlanPhase:
		return &m.PlanningOpts().DependencyLocks
	case ApplyPhase:
		return &m.applying.opts.DependencyLocks
	default:
		return nil

	}
}
