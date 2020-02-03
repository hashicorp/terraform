// Package backend provides interfaces that the CLI uses to interact with
// Terraform. A backend provides the abstraction that allows the same CLI
// to simultaneously support both local and remote operations for seamlessly
// using Terraform in a team environment.
package backend

import (
	"context"
	"errors"
	"time"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/command/clistate"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configload"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/plans/planfile"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statemgr"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// DefaultStateName is the name of the default, initial state that every
// backend must have. This state cannot be deleted.
const DefaultStateName = "default"

var (
	// ErrDefaultWorkspaceNotSupported is returned when an operation does not
	// support using the default workspace, but requires a named workspace to
	// be selected.
	ErrDefaultWorkspaceNotSupported = errors.New("default workspace not supported\n" +
		"You can create a new workspace with the \"workspace new\" command.")

	// ErrOperationNotSupported is returned when an unsupported operation
	// is detected by the configured backend.
	ErrOperationNotSupported = errors.New("operation not supported")

	// ErrWorkspacesNotSupported is an error returned when a caller attempts
	// to perform an operation on a workspace other than "default" for a
	// backend that doesn't support multiple workspaces.
	//
	// The caller can detect this to do special fallback behavior or produce
	// a specific, helpful error message.
	ErrWorkspacesNotSupported = errors.New("workspaces not supported")
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

	// PrepareConfig checks the validity of the values in the given
	// configuration, and inserts any missing defaults, assuming that its
	// structure has already been validated per the schema returned by
	// ConfigSchema.
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
	PrepareConfig(cty.Value) (cty.Value, tfdiags.Diagnostics)

	// Configure uses the provided configuration to set configuration fields
	// within the backend.
	//
	// The given configuration is assumed to have already been validated
	// against the schema returned by ConfigSchema and passed validation
	// via PrepareConfig.
	//
	// This method may be called only once per backend instance, and must be
	// called before all other methods except where otherwise stated.
	//
	// If error diagnostics are returned, the internal state of the instance
	// is undefined and no other methods may be called.
	Configure(cty.Value) tfdiags.Diagnostics

	// StateMgr returns the state manager for the given workspace name.
	//
	// If the returned state manager also implements statemgr.Locker then
	// it's the caller's responsibility to call Lock and Unlock as appropriate.
	//
	// If the named workspace doesn't exist, or if it has no state, it will
	// be created either immediately on this call or the first time
	// PersistState is called, depending on the state manager implementation.
	StateMgr(workspace string) (statemgr.Full, error)

	// DeleteWorkspace removes the workspace with the given name if it exists.
	//
	// DeleteWorkspace cannot prevent deleting a state that is in use. It is
	// the responsibility of the caller to hold a Lock for the state manager
	// belonging to this workspace before calling this method.
	DeleteWorkspace(name string) error

	// States returns a list of the names of all of the workspaces that exist
	// in this backend.
	Workspaces() ([]string, error)
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
	Context(*Operation) (*terraform.Context, statemgr.Full, tfdiags.Diagnostics)
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
	PlanOutBackend *plans.Backend

	// ConfigDir is the path to the directory containing the configuration's
	// root module.
	ConfigDir string

	// ConfigLoader is a configuration loader that can be used to load
	// configuration from ConfigDir.
	ConfigLoader *configload.Loader

	// Plan is a plan that was passed as an argument. This is valid for
	// plan and apply arguments but may not work for all backends.
	PlanFile *planfile.Reader

	// The options below are more self-explanatory and affect the runtime
	// behavior of the operation.
	AutoApprove  bool
	Destroy      bool
	DestroyForce bool
	Parallelism  int
	Targets      []addrs.Targetable
	Variables    map[string]UnparsedVariableValue

	// Some operations use root module variables only opportunistically or
	// don't need them at all. If this flag is set, the backend must treat
	// all variables as optional and provide an unknown value for any required
	// variables that aren't set in order to allow partial evaluation against
	// the resulting incomplete context.
	//
	// This flag is honored only if PlanFile isn't set. If PlanFile is set then
	// the variables set in the plan are used instead, and they must be valid.
	AllowUnsetVariables bool

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

	// PlanEmpty is populated after a Plan operation completes without error
	// to note whether a plan is empty or has changes.
	PlanEmpty bool

	// State is the final state after the operation completed. Persisting
	// this state is managed by the backend. This should only be read
	// after the operation completes to avoid read/write races.
	State *states.State
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
