// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package backendrun

import (
	"context"
	"log"

	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/planfile"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// HostAlias describes a list of aliases that should be used when initializing an
// [OperationsBackend].
type HostAlias struct {
	From svchost.Hostname
	To   svchost.Hostname
}

// OperationsBackend is an extension of [backend.Backend] for the few backends
// that can directly perform Terraform operations.
//
// Most backends are used only for remote state storage, and those should not
// implement this interface or import anything from this package.
type OperationsBackend interface {
	backend.Backend

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
	PlanFile *planfile.WrappedPlanFile

	// The options below are more self-explanatory and affect the runtime
	// behavior of the operation.
	PlanMode             plans.Mode
	AutoApprove          bool
	Targets              []addrs.Targetable
	ForceReplace         []addrs.AbsResourceInstance
	Variables            map[string]UnparsedVariableValue
	StatePersistInterval int

	// Some operations use root module variables only opportunistically or
	// don't need them at all. If this flag is set, the backend must treat
	// all variables as optional and provide an unknown value for any required
	// variables that aren't set in order to allow partial evaluation against
	// the resulting incomplete context.
	//
	// This flag is honored only if PlanFile isn't set. If PlanFile is set then
	// the variables set in the plan are used instead, and they must be valid.
	AllowUnsetVariables bool

	// DeferralAllowed enables experimental support for automatically performing
	// a partial plan if some objects are not yet plannable.
	//
	// IMPORTANT: When configuring an Operation, you should only set a value for
	// this field if Terraform was built with experimental features enabled.
	DeferralAllowed bool

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
