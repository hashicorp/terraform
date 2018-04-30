// Package backend provides interfaces that the CLI uses to interact with
// Terraform. A backend provides the abstraction that allows the same CLI
// to simultaneously support both local and remote operations for seamlessly
// using Terraform in a team environment.
package backend

import (
	"context"
	"errors"
	"time"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/command/clistate"
	"github.com/hashicorp/terraform/config/configschema"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configload"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
)

// DefaultStateName is the name of the default, initial state that every
// backend must have. This state cannot be deleted.
const DefaultStateName = "default"

// This must be returned rather than a custom error so that the Terraform
// CLI can detect it and handle it appropriately.
var (
	// ErrDefaultStateNotSupported is returned when an operation does not support
	// using the default state, but requires a named state to be selected.
	ErrDefaultStateNotSupported = errors.New("default state not supported\n" +
		"You can create a new workspace with the \"workspace new\" command.")

	// ErrNamedStatesNotSupported is returned when a named state operation
	// isn't supported.
	ErrNamedStatesNotSupported = errors.New("named states not supported")

	// ErrOperationNotSupported is returned when an unsupported operation
	// is detected by the configured backend.
	ErrOperationNotSupported = errors.New("operation not supported")
)

// InitFn is used to initialize a new backend.
type InitFn func() Backend

// Backend is the minimal interface that must be implemented to enable Terraform.
type Backend interface {
	// ConfigSchema returns a description of the expected configuration
	// structure for the receiving backend.
	//
	// This method does not have any side-effects for the backend and can
	// be safely used before configuring.
	ConfigSchema() *configschema.Block

	// ValidateConfig checks the validity of the values in the given
	// configuration, assuming that its structure has already been validated
	// per the schema returned by ConfigSchema.
	//
	// This method does not have any side-effects for the backend and can
	// be safely used before configuring. It also does not consult any
	// external data such as environment variables, disk files, etc. Validation
	// that requires such external data should be deferred until the
	// Configure call.
	//
	// If error diagnostics are returned then the configuration is not valid
	// and must not subsequently be passed to the Configure method.
	//
	// This method may return configuration-contextual diagnostics such
	// as tfdiags.AttributeValue, and so the caller should provide the
	// necessary context via the diags.InConfigBody method before returning
	// diagnostics to the user.
	ValidateConfig(cty.Value) tfdiags.Diagnostics

	// Configure uses the provided configuration to set configuration fields
	// within the backend.
	//
	// The given configuration is assumed to have already been validated
	// against the schema returned by ConfigSchema and passed validation
	// via ValidateConfig.
	//
	// This method may be called only once per backend instance, and must be
	// called before all other methods except where otherwise stated.
	//
	// If error diagnostics are returned, the internal state of the instance
	// is undefined and no other methods may be called.
	Configure(cty.Value) tfdiags.Diagnostics

	// State returns the current state for this environment. This state may
	// not be loaded locally: the proper APIs should be called on state.State
	// to load the state. If the state.State is a state.Locker, it's up to the
	// caller to call Lock and Unlock as needed.
	//
	// If the named state doesn't exist it will be created. The "default" state
	// is always assumed to exist.
	State(name string) (state.State, error)

	// DeleteState removes the named state if it exists. It is an error
	// to delete the default state.
	//
	// DeleteState does not prevent deleting a state that is in use. It is the
	// responsibility of the caller to hold a Lock on the state when calling
	// this method.
	DeleteState(name string) error

	// States returns a list of configured named states.
	States() ([]string, error)
}

// Enhanced implements additional behavior on top of a normal backend.
//
// Enhanced backends allow customizing the behavior of Terraform operations.
// This allows Terraform to potentially run operations remotely, load
// configurations from external sources, etc.
type Enhanced interface {
	Backend

	// Operation performs a Terraform operation such as refresh, plan, apply.
	// It is up to the implementation to determine what "performing" means.
	// This DOES NOT BLOCK. The context returned as part of RunningOperation
	// should be used to block for completion.
	// If the state used in the operation can be locked, it is the
	// responsibility of the Backend to lock the state for the duration of the
	// running operation.
	Operation(context.Context, *Operation) (*RunningOperation, error)
}

// Local implements additional behavior on a Backend that allows local
// operations in addition to remote operations.
//
// This enables more behaviors of Terraform that require more data such
// as `console`, `import`, `graph`. These require direct access to
// configurations, variables, and more. Not all backends may support this
// so we separate it out into its own optional interface.
type Local interface {
	// Context returns a runnable terraform Context. The operation parameter
	// doesn't need a Type set but it needs other options set such as Module.
	Context(*Operation) (*terraform.Context, state.State, tfdiags.Diagnostics)
}

// An operation represents an operation for Terraform to execute.
//
// Note that not all fields are supported by all backends and can result
// in an error if set. All backend implementations should show user-friendly
// errors explaining any incorrectly set values. For example, the local
// backend doesn't support a PlanId being set.
//
// The operation options are purposely designed to have maximal compatibility
// between Terraform and Terraform Servers (a commercial product offered by
// HashiCorp). Therefore, it isn't expected that other implementation support
// every possible option. The struct here is generalized in order to allow
// even partial implementations to exist in the open, without walling off
// remote functionality 100% behind a commercial wall. Anyone can implement
// against this interface and have Terraform interact with it just as it
// would with HashiCorp-provided Terraform Servers.
type Operation struct {
	// Type is the operation to perform.
	Type OperationType

	// PlanId is an opaque value that backends can use to execute a specific
	// plan for an apply operation.
	//
	// PlanOutBackend is the backend to store with the plan. This is the
	// backend that will be used when applying the plan.
	PlanId         string
	PlanRefresh    bool   // PlanRefresh will do a refresh before a plan
	PlanOutPath    string // PlanOutPath is the path to save the plan
	PlanOutBackend *terraform.BackendState

	// ConfigDir is the path to the directory containing the configuration's
	// root module.
	ConfigDir string

	// ConfigLoader is a configuration loader that can be used to load
	// configuration from ConfigDir.
	ConfigLoader *configload.Loader

	// Plan is a plan that was passed as an argument. This is valid for
	// plan and apply arguments but may not work for all backends.
	Plan *terraform.Plan

	// The options below are more self-explanatory and affect the runtime
	// behavior of the operation.
	AutoApprove  bool
	Destroy      bool
	Targets      []addrs.Targetable
	Variables    map[string]UnparsedVariableValue
	AutoApprove  bool
	DestroyForce bool

	// Input/output/control options.
	UIIn  terraform.UIInput
	UIOut terraform.UIOutput

	// If LockState is true, the Operation must Lock any
	// state.Lockers for its duration, and Unlock when complete.
	LockState bool

	// StateLocker is used to lock the state while providing UI feedback to the
	// user. This will be supplied by the Backend itself.
	StateLocker clistate.Locker

	// The duration to retry obtaining a State lock.
	StateLockTimeout time.Duration

	// Workspace is the name of the workspace that this operation should run
	// in, which controls which named state is used.
	Workspace string
}

// HasConfig returns true if and only if the operation has a ConfigDir value
// that refers to a directory containing at least one Terraform configuration
// file.
func (o *Operation) HasConfig() bool {
	return o.ConfigLoader.IsConfigDir(o.ConfigDir)
}

// Config loads the configuration that the operation applies to, using the
// ConfigDir and ConfigLoader fields within the receiving operation.
func (o *Operation) Config() (*configs.Config, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	config, hclDiags := o.ConfigLoader.LoadConfig(o.ConfigDir)
	diags = diags.Append(hclDiags)
	return config, diags
}

// RunningOperation is the result of starting an operation.
type RunningOperation struct {
	// For implementers of a backend, this context should not wrap the
	// passed in context. Otherwise, cancelling the parent context will
	// immediately mark this context as "done" but those aren't the semantics
	// we want: we want this context to be done only when the operation itself
	// is fully done.
	context.Context

	// Stop requests the operation to complete early, by calling Stop on all
	// the plugins. If the process needs to terminate immediately, call Cancel.
	Stop context.CancelFunc

	// Cancel is the context.CancelFunc associated with the embedded context,
	// and can be called to terminate the operation early.
	// Once Cancel is called, the operation should return as soon as possible
	// to avoid running operations during process exit.
	Cancel context.CancelFunc

	// Result is the exit status of the operation, populated only after the
	// operation has completed.
	Result OperationResult

	// ExitCode can be used to set a custom exit code. This enables enhanced
	// backends to set specific exit codes that miror any remote exit codes.
	ExitCode int

	// PlanEmpty is populated after a Plan operation completes without error
	// to note whether a plan is empty or has changes.
	PlanEmpty bool

	// State is the final state after the operation completed. Persisting
	// this state is managed by the backend. This should only be read
	// after the operation completes to avoid read/write races.
	State *terraform.State
}

// OperationResult describes the result status of an operation.
type OperationResult int

const (
	// OperationSuccess indicates that the operation completed as expected.
	OperationSuccess OperationResult = 0

	// OperationFailure indicates that the operation encountered some sort
	// of error, and thus may have been only partially performed or not
	// performed at all.
	OperationFailure OperationResult = 1
)

func (r OperationResult) ExitStatus() int {
	return int(r)
}
