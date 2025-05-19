// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package backend provides interfaces that the CLI uses to interact with
// Terraform. A backend provides the abstraction that allows the same CLI
// to simultaneously support both local and remote operations for seamlessly
// using Terraform in a team environment.
package backend

import (
	"context"
	"errors"
	"io/ioutil"
	"log"
	"os"

	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/planfile"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/mitchellh/go-homedir"
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
	DeleteWorkspace(name string, force bool) error

	// States returns a list of the names of all of the workspaces that exist
	// in this backend.
	Workspaces() ([]string, error)
}

// HostAlias describes a list of aliases that should be used when initializing an
// Enhanced Backend
type HostAlias struct {
	From svchost.Hostname
	To   svchost.Hostname
}

// Enhanced implements additional behavior on top of a normal backend.
//
// 'Enhanced' backends are an implementation detail only, and are no longer reflected as an external
// 'feature' of backends. In other words, backends refer to plugins for remote state snapshot
// storage only, and the Enhanced interface here is a necessary vestige of the 'local' and
// remote/cloud backends only.
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

	// ServiceDiscoveryAliases returns a mapping of Alias -> Target hosts to
	// configure.
	ServiceDiscoveryAliases() ([]HostAlias, error)
}

// Local implements additional behavior on a Backend that allows local
// operations in addition to remote operations.
//
// This enables more behaviors of Terraform that require more data such
// as `console`, `import`, `graph`. These require direct access to
// configurations, variables, and more. Not all backends may support this
// so we separate it out into its own optional interface.
type Local interface {
	// LocalRun uses information in the Operation to prepare a set of objects
	// needed to start running that operation.
	//
	// The operation doesn't need a Type set, but it needs various other
	// options set. This is a rather odd API that tries to treat all
	// operations as the same when they really aren't; see the local and remote
	// backend's implementations of this to understand what this actually
	// does, because this operation has no well-defined contract aside from
	// "whatever it already does".
	LocalRun(*Operation) (*LocalRun, statemgr.Full, tfdiags.Diagnostics)
}

// LocalRun represents the assortment of objects that we can collect or
// calculate from an Operation object, which we can then use for local
// operations.
//
// The operation methods on terraform.Context (Plan, Apply, Import, etc) each
// generate new artifacts which supersede parts of the LocalRun object that
// started the operation, so callers should be careful to use those subsequent
// artifacts instead of the fields of LocalRun where appropriate. The LocalRun
// data intentionally doesn't update as a result of calling methods on Context,
// in order to make data flow explicit.
//
// This type is a weird architectural wart resulting from the overly-general
// way our backend API models operations, whereby we behave as if all
// Terraform operations have the same inputs and outputs even though they
// are actually all rather different. The exact meaning of the fields in
// this type therefore vary depending on which OperationType was passed to
// Local.Context in order to create an object of this type.
type LocalRun struct {
	// Core is an already-initialized Terraform Core context, ready to be
	// used to run operations such as Plan and Apply.
	Core *terraform.Context

	// Config is the configuration we're working with, which typically comes
	// from either config files directly on local disk (when we're creating
	// a plan, or similar) or from a snapshot embedded in a plan file
	// (when we're applying a saved plan).
	Config *configs.Config

	// InputState is the state that should be used for whatever is the first
	// method call to a context created with CoreOpts. When creating a plan
	// this will be the previous run state, but when applying a saved plan
	// this will be the prior state recorded in that plan.
	InputState *states.State

	// PlanOpts are options to pass to a Plan or Plan-like operation.
	//
	// This is nil when we're applying a saved plan, because the plan itself
	// contains enough information about its options to apply it.
	PlanOpts *terraform.PlanOpts

	// Plan is a plan loaded from a saved plan file, if our operation is to
	// apply that saved plan.
	//
	// This is nil when we're not applying a saved plan.
	Plan *plans.Plan
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

	// DependencyLocks represents the locked dependencies associated with
	// the configuration directory given in ConfigDir.
	//
	// Note that if field PlanFile is set then the plan file should contain
	// its own dependency locks. The backend is responsible for correctly
	// selecting between these two sets of locks depending on whether it
	// will be using ConfigDir or PlanFile to get the configuration for
	// this operation.
	DependencyLocks *depsfile.Locks

	// Hooks can be used to perform actions triggered by various events during
	// the operation's lifecycle.
	Hooks []terraform.Hook

	// Plan is a plan that was passed as an argument. This is valid for
	// plan and apply arguments but may not work for all backends.
	PlanFile *planfile.Reader

	// The options below are more self-explanatory and affect the runtime
	// behavior of the operation.
	PlanMode     plans.Mode
	AutoApprove  bool
	Targets      []addrs.Targetable
	ForceReplace []addrs.AbsResourceInstance
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

	// View implements the logic for all UI interactions.
	View views.Operation

	// Input/output/control options.
	UIIn  terraform.UIInput
	UIOut terraform.UIOutput

	// StateLocker is used to lock the state while providing UI feedback to the
	// user. This will be replaced by the Backend to update the context.
	//
	// If state locking is not necessary, this should be set to a no-op
	// implementation of clistate.Locker.
	StateLocker clistate.Locker

	// Workspace is the name of the workspace that this operation should run
	// in, which controls which named state is used.
	Workspace string

	// GenerateConfigOut tells the operation both that it should generate config
	// for unmatched import targets and where any generated config should be
	// written to.
	GenerateConfigOut string
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

// ReportResult is a helper for the common chore of setting the status of
// a running operation and showing any diagnostics produced during that
// operation.
//
// If the given diagnostics contains errors then the operation's result
// will be set to backend.OperationFailure. It will be set to
// backend.OperationSuccess otherwise. It will then use o.View.Diagnostics
// to show the given diagnostics before returning.
//
// Callers should feel free to do each of these operations separately in
// more complex cases where e.g. diagnostics are interleaved with other
// output, but terminating immediately after reporting error diagnostics is
// common and can be expressed concisely via this method.
func (o *Operation) ReportResult(op *RunningOperation, diags tfdiags.Diagnostics) {
	if diags.HasErrors() {
		op.Result = OperationFailure
	} else {
		op.Result = OperationSuccess
	}
	if o.View != nil {
		o.View.Diagnostics(diags)
	} else {
		// Shouldn't generally happen, but if it does then we'll at least
		// make some noise in the logs to help us spot it.
		if len(diags) != 0 {
			log.Printf(
				"[ERROR] Backend needs to report diagnostics but View is not set:\n%s",
				diags.ErrWithWarnings(),
			)
		}
	}
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

	// PlanEmpty is populated after a Plan operation completes to note whether
	// a plan is empty or has changes. This is only used in the CLI to determine
	// the exit status because the plan value is not available at that point.
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

// If the argument is a path, Read loads it and returns the contents,
// otherwise the argument is assumed to be the desired contents and is simply
// returned.
func ReadPathOrContents(poc string) (string, error) {
	if len(poc) == 0 {
		return poc, nil
	}

	path := poc
	if path[0] == '~' {
		var err error
		path, err = homedir.Expand(path)
		if err != nil {
			return path, err
		}
	}

	if _, err := os.Stat(path); err == nil {
		contents, err := ioutil.ReadFile(path)
		if err != nil {
			return string(contents), err
		}
		return string(contents), nil
	}

	return poc, nil
}
