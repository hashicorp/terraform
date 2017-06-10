---
layout: "commands-workspace"
page_title: "Command: workspace new"
sidebar_current: "docs-workspace-sub-new"
description: |-
  The terraform workspace new command is used to create a new workspace.
---

# Command: workspace new

The `terraform workspace new` command is used to create a new workspace.

## Usage

Usage: `terraform workspace new [NAME]`

This command will create a new workspace with the given name. A workspace with
this name must not already exist.

If the `-state` flag is given, the state specified by the given path
will be copied to initialize the state for this new workspace.

The command-line flags are all optional. The only supported flag is:

* `-state=path` - Path to a state file to initialize the state of this environment.

## Example: Create

```
$ terraform workspace new example
Created and switched to workspace "example"!

You're now on a new, empty workspace. Workspaces isolate their state,
so if you run "terraform plan" Terraform will not see any existing state
for this configuration.
```

## Example: Create from State

To create a new workspace from a pre-existing local state file:

```
$ terraform workspace new -state=old.terraform.tfstate example
Created and switched to workspace "example".

You're now on a new, empty workspace. Workspaces isolate their state,
so if you run "terraform plan" Terraform will not see any existing state
for this configuration.
```
