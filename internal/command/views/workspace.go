// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

const EnvCreated = `
[reset][green][bold]Created and switched to workspace %q![reset][green]

You're now on a new, empty workspace. Workspaces isolate their state,
so if you run "terraform plan" Terraform will not see any existing state
for this configuration.
`

const envCreatedWithoutStatePopulation = `
[reset][yellow][bold]Created and switched to workspace %q, but failed to initialize the state.[reset][yellow]

You're now on a new, empty workspace. However, errors prevented Terraform from initializing the new workspace's state
using the provided '-state' flag. You will need to address these errors before the workspace's state can be properly
initialized. Once they're addressed, you can use "terraform workspace delete %s" to delete the newly created
workspace and then reattempt this operation.
`
