// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// errWrongWorkspaceForPlan is a custom error used to alert users that the plan file they are applying
// describes a workspace that doesn't match the currently selected workspace.
//
// This needs to render slightly different errors depending on whether we're using:
// > CE Workspaces (remote-state backends, local backends)
// > HCP Terraform Workspaces (cloud backend)
type errWrongWorkspaceForPlan struct {
	plannedWorkspace string
	currentWorkspace string
	isCloud          bool
}

func (e *errWrongWorkspaceForPlan) Error() string {
	msg := fmt.Sprintf(`The plan file describes changes to the %q workspace, but the %q workspace is currently in use.

Applying this plan with the incorrect workspace selected could result in state being stored in an unexpected location, or a downstream error when Terraform attempts apply a plan using the other workspace's state.`,
		e.plannedWorkspace,
		e.currentWorkspace,
	)

	// For users to understand what's happened and how to correct it we'll give some guidance,
	// but that guidance depends on whether a cloud backend is in use or not.
	if e.isCloud {
		// When using the cloud backend the solution is to focus on the cloud block and running init
		msg = msg + fmt.Sprintf(` If you'd like to continue to use the plan file, make sure the cloud block in your configuration contains the workspace name %q.
In future, make sure your cloud block is correct and unchanged since the last time you performed "terraform init" before creating a plan.`, e.plannedWorkspace)
	} else {
		// When using the backend block the solution is to not select a different workspace
		// between plan and apply operations.
		msg = msg + fmt.Sprintf(` If you'd like to continue to use the plan file, you must run "terraform workspace select %s" to select the matching workspace.
In future make sure the selected workspace is not changed between creating and applying a plan file.
`, e.plannedWorkspace)
	}

	return msg
}

// errBackendLocalRead is a custom error used to alert users that state
// files on their local filesystem were not erased successfully after
// migrating that state to a remote-state backend.
type errBackendLocalRead struct {
	innerError error
}

func (e *errBackendLocalRead) Error() string {
	return fmt.Sprintf(`Error reading local state: %s

Terraform is trying to read your local state to determine if there is
state to migrate to your newly configured backend. Terraform can't continue
without this check because that would risk losing state. Please resolve the
error above and try again.`, e.innerError)
}

// errBackendMigrateLocalDelete is a custom error used to alert users that state
// files on their local filesystem were not erased successfully after migrating
// that state to a remote-state backend.
type errBackendMigrateLocalDelete struct {
	innerError error
}

func (e *errBackendMigrateLocalDelete) Error() string {
	return fmt.Sprintf(`Error deleting local state after migration: %s

Your local state is deleted after successfully migrating it to the newly
configured backend. As part of the deletion process, a backup is made at
the standard backup path unless explicitly asked not to. To cleanly operate
with a backend, we must delete the local state file. Please resolve the
issue above and retry the command.`, e.innerError)
}

// errBackendSavedUnknown is a custom error used to alert users that their
// configuration describes a backend that's not implemented in Terraform.
type errBackendNewUnknown struct {
	backendName string
}

func (e *errBackendNewUnknown) Error() string {
	return fmt.Sprintf(`The backend %q could not be found.

This is the backend specified in your Terraform configuration file.
This error could be a simple typo in your configuration, but it can also
be caused by using a Terraform version that doesn't support the specified
backend type. Please check your configuration and your Terraform version.

If you'd like to run Terraform and store state locally, you can fix this
error by removing the backend configuration from your configuration.`, e.backendName)
}

// errBackendSavedUnknown is a custom error used to alert users that their
// plan file describes a backend that's not implemented in Terraform.
type errBackendSavedUnknown struct {
	backendName string
}

func (e *errBackendSavedUnknown) Error() string {
	return fmt.Sprintf(`The backend %q could not be found.

This is the backend that this Terraform environment is configured to use
both in your configuration and saved locally as your last-used backend.
If it isn't found, it could mean an alternate version of Terraform was
used with this configuration. Please use the proper version of Terraform that
contains support for this backend.

If you'd like to force remove this backend, you must update your configuration
to not use the backend and run "terraform init" (or any other command) again.`, e.backendName)
}

// errBackendClearSaved is a custom error used to alert users that
// Terraform failed to empty the backend state file's contents.
type errBackendClearSaved struct {
	innerError error
}

func (e *errBackendClearSaved) Error() string {
	return fmt.Sprintf(`Error clearing the backend configuration: %s

Terraform removes the saved backend configuration when you're removing a
configured backend. This must be done so future Terraform runs know to not
use the backend configuration. Please look at the error above, resolve it,
and try again.`, e.innerError)
}

// errBackendInitDiag creates a diagnostic to present to users when
// users attempt to run a non-init command after making a change to their
// backend configuration.
//
// An init reason should be provided as an argument.
func errBackendInitDiag(initReason string) tfdiags.Diagnostic {
	msg := fmt.Sprintf(`Reason: %s

The "backend" is the interface that Terraform uses to store state,
perform operations, etc. If this message is showing up, it means that the
Terraform configuration you're using is using a custom configuration for
the Terraform backend.

Changes to backend configurations require reinitialization. This allows
Terraform to set up the new configuration, copy existing state, etc. Please run
"terraform init" with either the "-reconfigure" or "-migrate-state" flags to
use the current configuration.

If the change reason above is incorrect, please verify your configuration
hasn't changed and try again. At this point, no changes to your existing
configuration or state have been made.`, initReason)

	return tfdiags.Sourceless(
		tfdiags.Error,
		"Backend initialization required, please run \"terraform init\"",
		msg,
	)
}

// errStateStoreInitDiag creates a diagnostic to present to users when
// users attempt to run a non-init command after making a change to their
// state_store configuration.
//
// An init reason should be provided as an argument.
func errStateStoreInitDiag(initReason string) tfdiags.Diagnostic {
	msg := fmt.Sprintf(`Reason: %s

The "state store" is the interface that Terraform uses to store state when
performing operations on the local machine. If this message is showing up,
it means that the Terraform configuration you're using is using a custom
configuration for state storage in Terraform.

Changes to state store configurations require reinitialization. This allows
Terraform to set up the new configuration, copy existing state, etc. Please run
"terraform init" with either the "-reconfigure" or "-migrate-state" flags to
use the current configuration.

If the change reason above is incorrect, please verify your configuration
hasn't changed and try again. At this point, no changes to your existing
configuration or state have been made.`, initReason)

	return tfdiags.Sourceless(
		tfdiags.Error,
		"State store initialization required, please run \"terraform init\"",
		msg,
	)
}

// errBackendInitCloudDiag creates a diagnostic to present to users when
// an init command encounters config changes in a `cloud` block.
//
// An init reason should be provided as an argument.
func errBackendInitCloudDiag(initReason string) tfdiags.Diagnostic {
	msg := fmt.Sprintf(`Reason: %s.

Changes to the HCP Terraform configuration block require reinitialization, to discover any changes to the available workspaces.

To re-initialize, run:
  terraform init

Terraform has not yet made changes to your existing configuration or state.`, initReason)

	return tfdiags.Sourceless(
		tfdiags.Error,
		"HCP Terraform or Terraform Enterprise initialization required: please run \"terraform init\"",
		msg,
	)
}

// errBackendWriteSavedDiag creates a diagnostic to present to users when
// an init command experiences an error while writing to the backend state file.
func errBackendWriteSavedDiag(innerError error) tfdiags.Diagnostic {
	msg := fmt.Sprintf(`Error saving the backend configuration: %s

Terraform saves the complete backend configuration in a local file for
configuring the backend on future operations. This cannot be disabled. Errors
are usually due to simple file permission errors. Please look at the error
above, resolve it, and try again.`, innerError)

	return tfdiags.Sourceless(
		tfdiags.Error,
		"Backend initialization failed",
		msg,
	)
}

// errBackendNoExistingWorkspaces is returned by calling code when it expects a backend.Backend
// to report one or more workspaces exist.
//
// The returned error may be used as a sentinel error and acted upon or just wrapped in a
// diagnostic and returned.
type errBackendNoExistingWorkspaces struct{}

func (e *errBackendNoExistingWorkspaces) Error() string {
	return `No existing workspaces.

Use the "terraform workspace" command to create and select a new workspace.
If the backend already contains existing workspaces, you may need to update
the backend configuration.`
}

func errStateStoreWorkspaceCreateDiag(innerError error, storeType string) tfdiags.Diagnostic {
	msg := fmt.Sprintf(`Error creating the default workspace using pluggable state store %s: %s

This could be a bug in the provider used for state storage, or a bug in
Terraform. Please file an issue with the provider developers before reporting
a bug for Terraform.`,
		storeType,
		innerError,
	)

	return tfdiags.Sourceless(
		tfdiags.Error,
		"Cannot create the default workspace",
		msg,
	)
}

// migrateOrReconfigDiag creates a diagnostic to present to users when
// an init command encounters a mismatch in backend state and the current config
// and Terraform needs users to provide additional instructions about how Terraform
// should proceed.
var migrateOrReconfigDiag = tfdiags.Sourceless(
	tfdiags.Error,
	"Backend configuration changed",
	"A change in the backend configuration has been detected, which may require migrating existing state.\n\n"+
		"If you wish to attempt automatic migration of the state, use \"terraform init -migrate-state\".\n"+
		`If you wish to store the current configuration with no changes to the state, use "terraform init -reconfigure".`)

// migrateOrReconfigStateStoreDiag creates a diagnostic to present to users when
// an init command encounters a mismatch in state store config state and the current config
// and Terraform needs users to provide additional instructions about how it
// should proceed.
var migrateOrReconfigStateStoreDiag = tfdiags.Sourceless(
	tfdiags.Error,
	"State store configuration changed",
	"A change in the state store configuration has been detected, which may require migrating existing state.\n\n"+
		"If you wish to attempt automatic migration of the state, use \"terraform init -migrate-state\".\n"+
		`If you wish to store the current configuration with no changes to the state, use "terraform init -reconfigure".`)

// errStateStoreClearSaved is a custom error used to alert users that
// Terraform failed to empty the state store state file's contents.
type errStateStoreClearSaved struct {
	innerError error
}

func (e *errStateStoreClearSaved) Error() string {
	return fmt.Sprintf(`Error clearing the state store configuration: %s

Terraform removes the saved state store configuration when you're removing a
configured state store. This must be done so future Terraform runs know to not
use the state store configuration. Please look at the error above, resolve it,
and try again.`, e.innerError)
}
