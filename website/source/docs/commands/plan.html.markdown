---
layout: "docs"
page_title: "Command: plan"
sidebar_current: "docs-commands-plan"
description: |-
  The `terraform plan` command is used to create an execution plan. Terraform performs a refresh, unless explicitly disabled, and then determines what actions are necessary to achieve the desired state specified in the configuration files. The plan can be saved using `-out`, and then provided to `terraform apply` to ensure only the pre-planned actions are executed.
---

# Command: plan

The `terraform plan` command is used to create an execution plan. Terraform
performs a refresh, unless explicitly disabled, and then determines what
actions are necessary to achieve the desired state specified in the
configuration files. The plan can be saved using `-out`, and then provided
to `terraform apply` to ensure only the pre-planned actions are executed.

## Usage

Usage: `terraform plan [options] [dir]`

By default, `plan` requires no flags and looks in the current directory
for the configuration and state file to refresh.

The command-line flags are all optional. The list of available flags are:

* `-backup=path` - Path to the backup file. Defaults to `-state-out` with
  the ".backup" extension. Disabled by setting to "-".

* `-destroy` - If set, generates a plan to destroy all the known resources.

* `-detailed-exitcode` - Return a detailed exit code when the command exits.
  When provided, this argument changes the exit codes and their meanings to
  provide more granular information about what the resulting plan contains:
  * 0 = Succeeded with empty diff (no changes)
  * 1 = Error
  * 2 = Succeeded with non-empty diff (changes present)

* `-input=true` - Ask for input for variables if not directly set.

* `-module-depth=n` - Specifies the depth of modules to show in the output.
  This does not affect the plan itself, only the output shown. By default,
  this is -1, which will expand all.

* `-no-color` - Disables output with coloring.

* `-out=path` - The path to save the generated execution plan. This plan
  can then be used with `terraform apply` to be certain that only the
  changes shown in this plan are applied. Read the warning on saved
  plans below.

* `-parallelism=n` - Limit the number of concurrent operation as Terraform
  [walks the graph](/docs/internals/graph.html#walking-the-graph).

* `-refresh=true` - Update the state prior to checking for differences.

* `-state=path` - Path to the state file. Defaults to "terraform.tfstate".

* `-target=resource` - A [Resource
  Address](/docs/internals/resource-addressing.html) to target. Operation will
  be limited to this resource and its dependencies. This flag can be used
  multiple times.

* `-var 'foo=bar'` - Set a variable in the Terraform configuration. This
  flag can be set multiple times.

* `-var-file=foo` - Set variables in the Terraform configuration from
   a file. If "terraform.tfvars" is present, it will be automatically
   loaded if this flag is not specified. This flag can be used multiple times.

## Security Warning

Saved plan files (with the `-out` flag) encode the configuration,
state, diff, and _variables_. Variables are often used to store secrets.
Therefore, the plan file can potentially store secrets.

Terraform itself does not encrypt the plan file. It is highly
recommended to encrypt the plan file if you intend to transfer it
or keep it at rest for an extended period of time.

Future versions of Terraform will make plan files more
secure.
