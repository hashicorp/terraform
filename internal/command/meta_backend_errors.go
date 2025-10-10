// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import "fmt"

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

type errBackendInit struct {
	initReason string
}

func (e *errBackendInit) Error() string {
	return fmt.Sprintf(`Reason: %s

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
configuration or state have been made.`, e.initReason)
}

type errBackendInitCloud struct {
	initReason string
}

func (e *errBackendInitCloud) Error() string {
	return fmt.Sprintf(`Reason: %s.

Changes to the HCP Terraform configuration block require reinitialization, to discover any changes to the available workspaces.

To re-initialize, run:
  terraform init

Terraform has not yet made changes to your existing configuration or state.`, e.initReason)
}

type errBackendWriteSaved struct {
	innerError error
}

func (e *errBackendWriteSaved) Error() string {
	return fmt.Sprintf(`Error saving the backend configuration: %s

Terraform saves the complete backend configuration in a local file for
configuring the backend on future operations. This cannot be disabled. Errors
are usually due to simple file permission errors. Please look at the error
above, resolve it, and try again.`, e.innerError)
}

type errBackendNoExistingWorkspaces struct{}

func (e *errBackendNoExistingWorkspaces) Error() string {
	return `No existing workspaces.

Use the "terraform workspace" command to create and select a new workspace.
If the backend already contains existing workspaces, you may need to update
the backend configuration.`
}
