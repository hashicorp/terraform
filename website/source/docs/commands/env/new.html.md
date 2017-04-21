---
layout: "commands-env"
page_title: "Command: env new"
sidebar_current: "docs-env-sub-new"
description: |-
  The terraform env new command is used to create a new state environment.
---

# Command: env new

The `terraform env new` command is used to create a new state
environment.

## Usage

Usage: `terraform env new [NAME]`

This command will create a new environment with the given name. This
environment must not already exist.

If the `-state` flag is given, the state specified by the given path
will be copied to initialize the state for this new environment.

The command-line flags are all optional. The list of available flags are:

* `-state=path` - Path to a state file to initialize the state of this environment.

## Example: Create

```
$ terraform env new example
Created and switched to environment "example"!

You're now on a new, empty environment. Environments isolate their state,
so if you run "terraform plan" Terraform will not see any existing state
for this configuration.
```

## Example: Create from State

To create a new environment from a pre-existing state path:

```
$ terraform env new -state=old.terraform.tfstate example
Created and switched to environment "example"!

You're now on a new, empty environment. Environments isolate their state,
so if you run "terraform plan" Terraform will not see any existing state
for this configuration.
```
