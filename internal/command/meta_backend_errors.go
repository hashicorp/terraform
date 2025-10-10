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

type errBackendNoExistingWorkspaces struct{}

func (e *errBackendNoExistingWorkspaces) Error() string {
	return `No existing workspaces.

Use the "terraform workspace" command to create and select a new workspace.
If the backend already contains existing workspaces, you may need to update
the backend configuration.`
}
