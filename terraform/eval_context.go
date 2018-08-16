package terraform

import (
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/lang"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// EvalContext is the interface that is given to eval nodes to execute.
type EvalContext interface {
	// Stopped returns a channel that is closed when evaluation is stopped
	// via Terraform.Context.Stop()
	Stopped() <-chan struct{}

	// Path is the current module path.
	Path() addrs.ModuleInstance

	// Hook is used to call hook methods. The callback is called for each
	// hook and should return the hook action to take and the error.
	Hook(func(Hook) (HookAction, error)) error

	// Input is the UIInput object for interacting with the UI.
	Input() UIInput

	// InitProvider initializes the provider with the given type and address, and
	// returns the implementation of the resource provider or an error.
	//
	// It is an error to initialize the same provider more than once.
	InitProvider(typ string, addr addrs.ProviderConfig) (providers.Interface, error)

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
	ProviderSchema(addrs.AbsProviderConfig) *ProviderSchema

	// CloseProvider closes provider connections that aren't needed anymore.
	CloseProvider(addrs.ProviderConfig) error

	// ConfigureProvider configures the provider with the given
	// configuration. This is a separate context call because this call
	// is used to store the provider configuration for inheritance lookups
	// with ParentProviderConfig().
	ConfigureProvider(addrs.ProviderConfig, cty.Value) tfdiags.Diagnostics

	// ProviderInput and SetProviderInput are used to configure providers
	// from user input.
	ProviderInput(addrs.ProviderConfig) map[string]cty.Value
	SetProviderInput(addrs.ProviderConfig, map[string]cty.Value)

	// InitProvisioner initializes the provisioner with the given name and
	// returns the implementation of the resource provisioner or an error.
	//
	// It is an error to initialize the same provisioner more than once.
	InitProvisioner(string) (ResourceProvisioner, error)

	// Provisioner gets the provisioner instance with the given name (already
	// initialized) or returns nil if the provisioner isn't initialized.
	Provisioner(string) ResourceProvisioner

	// ProvisionerSchema retrieves the main configuration schema for a
	// particular provisioner, which must have already been initialized with
	// InitProvisioner.
	ProvisionerSchema(string) *configschema.Block

	// CloseProvisioner closes provisioner connections that aren't needed
	// anymore.
	CloseProvisioner(string) error

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

	// EvaluationScope returns a scope that can be used to evaluate reference
	// addresses in this context.
	EvaluationScope(self addrs.Referenceable, keyData InstanceKeyEvalData) *lang.Scope

	// SetModuleCallArguments defines values for the variables of a particular
	// child module call.
	//
	// Calling this function multiple times has merging behavior, keeping any
	// previously-set keys that are not present in the new map.
	SetModuleCallArguments(addrs.ModuleCallInstance, map[string]cty.Value)

	// Changes returns the writer object that can be used to write new proposed
	// changes into the global changes set.
	Changes() *plans.ChangesSync

	// State returns a wrapper object that provides safe concurrent access to
	// the global state.
	State() *states.SyncState
}
