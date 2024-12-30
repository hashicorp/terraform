// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/experiments"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/moduletest/mocking"
	"github.com/hashicorp/terraform/internal/namedvals"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/deferring"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/provisioners"
	"github.com/hashicorp/terraform/internal/refactoring"
	"github.com/hashicorp/terraform/internal/resources/ephemeral"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type hookFunc func(func(Hook) (HookAction, error)) error

// EvalContext is the interface that is given to eval nodes to execute.
type EvalContext interface {
	// Stopped returns a context that is canceled when evaluation is stopped via
	// Terraform.Context.Stop()
	StopCtx() context.Context

	// Path is the current module path.
	Path() addrs.ModuleInstance

	// Hook is used to call hook methods. The callback is called for each
	// hook and should return the hook action to take and the error.
	Hook(func(Hook) (HookAction, error)) error

	// Input is the UIInput object for interacting with the UI.
	Input() UIInput

	// InitProvider initializes the provider with the given address, and returns
	// the implementation of the resource provider or an error.
	//
	// It is an error to initialize the same provider more than once. This
	// method will panic if the module instance address of the given provider
	// configuration does not match the Path() of the EvalContext.
	InitProvider(addr addrs.AbsProviderConfig, configs *configs.Provider) (providers.Interface, error)

	// Provider gets the provider instance with the given address (already
	// initialized) or returns nil if the provider isn't initialized.
	//
	// This method expects an _absolute_ provider configuration address, since
	// resources in one module are able to use providers from other modules.
	// InitProvider must've been called on the EvalContext of the module
	// that owns the given provider before calling this method.
	Provider(addrs.AbsProviderConfig) providers.Interface

	// ProviderSchema retrieves the schema for a particular provider, which
	// must have already been initialized with InitProvider.
	//
	// This method expects an _absolute_ provider configuration address, since
	// resources in one module are able to use providers from other modules.
	ProviderSchema(addrs.AbsProviderConfig) (providers.ProviderSchema, error)

	// CloseProvider closes provider connections that aren't needed anymore.
	//
	// This method will panic if the module instance address of the given
	// provider configuration does not match the Path() of the EvalContext.
	CloseProvider(addrs.AbsProviderConfig) error

	// ConfigureProvider configures the provider with the given
	// configuration. This is a separate context call because this call
	// is used to store the provider configuration for inheritance lookups
	// with ParentProviderConfig().
	//
	// This method will panic if the module instance address of the given
	// provider configuration does not match the Path() of the EvalContext.
	ConfigureProvider(addrs.AbsProviderConfig, cty.Value) tfdiags.Diagnostics

	// ProviderInput and SetProviderInput are used to configure providers
	// from user input.
	//
	// These methods will panic if the module instance address of the given
	// provider configuration does not match the Path() of the EvalContext.
	ProviderInput(addrs.AbsProviderConfig) map[string]cty.Value
	SetProviderInput(addrs.AbsProviderConfig, map[string]cty.Value)

	// Provisioner gets the provisioner instance with the given name.
	Provisioner(string) (provisioners.Interface, error)

	// ProvisionerSchema retrieves the main configuration schema for a
	// particular provisioner, which must have already been initialized with
	// InitProvisioner.
	ProvisionerSchema(string) (*configschema.Block, error)

	// ClosePlugins closes all cached provisioner and provider plugins.
	ClosePlugins() error

	// EvaluateBlock takes the given raw configuration block and associated
	// schema and evaluates it to produce a value of an object type that
	// conforms to the implied type of the schema.
	//
	// The "self" argument is optional. If given, it is the referenceable
	// address that the name "self" should behave as an alias for when
	// evaluating. Set this to nil if the "self" object should not be available.
	//
	// The "key" argument is also optional. If given, it is the instance key
	// of the current object within the multi-instance container it belongs
	// to. For example, on a resource block with "count" set this should be
	// set to a different addrs.IntKey for each instance created from that
	// block. Set this to addrs.NoKey if not appropriate.
	//
	// The returned body is an expanded version of the given body, with any
	// "dynamic" blocks replaced with zero or more static blocks. This can be
	// used to extract correct source location information about attributes of
	// the returned object value.
	EvaluateBlock(body hcl.Body, schema *configschema.Block, self addrs.Referenceable, keyData InstanceKeyEvalData) (cty.Value, hcl.Body, tfdiags.Diagnostics)

	// EvaluateExpr takes the given HCL expression and evaluates it to produce
	// a value.
	//
	// The "self" argument is optional. If given, it is the referenceable
	// address that the name "self" should behave as an alias for when
	// evaluating. Set this to nil if the "self" object should not be available.
	EvaluateExpr(expr hcl.Expression, wantType cty.Type, self addrs.Referenceable) (cty.Value, tfdiags.Diagnostics)

	// EvaluateReplaceTriggeredBy takes the raw reference expression from the
	// config, and returns the evaluated *addrs.Reference along with a boolean
	// indicating if that reference forces replacement.
	EvaluateReplaceTriggeredBy(expr hcl.Expression, repData instances.RepetitionData) (*addrs.Reference, bool, tfdiags.Diagnostics)

	// EvaluationScope returns a scope that can be used to evaluate reference
	// addresses in this context.
	EvaluationScope(self addrs.Referenceable, source addrs.Referenceable, keyData InstanceKeyEvalData) *lang.Scope

	// LanguageExperimentActive returns true if the given experiment is
	// active in the module associated with this EvalContext, or false
	// otherwise.
	LanguageExperimentActive(experiment experiments.Experiment) bool

	// EphemeralResources returns a helper object for tracking active
	// instances of ephemeral resources declared in the configuration.
	EphemeralResources() *ephemeral.Resources

	// NamedValues returns the object that tracks the gradual evaluation of
	// all input variables, local values, and output values during a graph
	// walk.
	NamedValues() *namedvals.State

	// Changes returns the writer object that can be used to write new proposed
	// changes into the global changes set.
	Changes() *plans.ChangesSync

	// State returns a wrapper object that provides safe concurrent access to
	// the global state.
	State() *states.SyncState

	// Checks returns the object that tracks the state of any custom checks
	// declared in the configuration.
	Checks() *checks.State

	// RefreshState returns a wrapper object that provides safe concurrent
	// access to the state used to store the most recently refreshed resource
	// values.
	RefreshState() *states.SyncState

	// PrevRunState returns a wrapper object that provides safe concurrent
	// access to the state which represents the result of the previous run,
	// updated only so that object data conforms to current schemas for
	// meaningful comparison with RefreshState.
	PrevRunState() *states.SyncState

	// InstanceExpander returns a helper object for tracking the expansion of
	// graph nodes during the plan phase in response to "count" and "for_each"
	// arguments.
	//
	// The InstanceExpander is a global object that is shared across all of the
	// EvalContext objects for a given configuration.
	InstanceExpander() *instances.Expander

	// Deferrals returns a helper object for tracking deferred actions, which
	// means that Terraform either cannot plan an action at all or cannot
	// perform a planned action due to an upstream dependency being deferred.
	Deferrals() *deferring.Deferred

	// MoveResults returns a map describing the results of handling any
	// resource instance move statements prior to the graph walk, so that
	// the graph walk can then record that information appropriately in other
	// artifacts produced by the graph walk.
	//
	// This data structure is created prior to the graph walk and read-only
	// thereafter, so callers must not modify the returned map or any other
	// objects accessible through it.
	MoveResults() refactoring.MoveResults

	// Overrides contains the modules and resources we should mock as part of
	// this execution.
	Overrides() *mocking.Overrides

	// withScope derives a new EvalContext that has all of the same global
	// context, but a new evaluation scope.
	withScope(scope evalContextScope) EvalContext

	// Forget if set to true will cause the plan to forget all resources. This is
	// only allowed in the context of a destroy plan.
	Forget() bool
}

func evalContextForModuleInstance(baseCtx EvalContext, addr addrs.ModuleInstance) EvalContext {
	return baseCtx.withScope(evalContextModuleInstance{
		Addr: addr,
	})
}
