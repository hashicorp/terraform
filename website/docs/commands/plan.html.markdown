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
configuration files.

This command is a convenient way to check whether the execution plan for a
set of changes matches your expectations without making any changes to
real resources or to the state. For example, `terraform plan` might be run
before committing a change to version control, to create confidence that it
will behave as expected.

The optional `-out` argument can be used to save the generated plan to a file
for later execution with `terraform apply`, which can be useful when
[running Terraform in automation](/guides/running-terraform-in-automation.html).

## Usage

Usage: `terraform plan [options] [dir-or-plan]`

By default, `plan` requires no flags and looks in the current directory
for the configuration and state file to refresh.

If the command is given an existing saved plan as an argument, the
command will output the contents of the saved plan. In this scenario,
the `plan` command will not modify the given plan. This can be used to
inspect a planfile.

The command-line flags are all optional. The list of available flags are:

* `-destroy` - If set, generates a plan to destroy all the known resources.

* `-detailed-exitcode` - Return a detailed exit code when the command exits.
  When provided, this argument changes the exit codes and their meanings to
  provide more granular information about what the resulting plan contains:
  * 0 = Succeeded with empty diff (no changes)
  * 1 = Error
  * 2 = Succeeded with non-empty diff (changes present)

* `-input=true` - Ask for input for variables if not directly set.

* `-lock=true` - Lock the state file when locking is supported.

* `-lock-timeout=0s` - Duration to retry a state lock.

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
  Ignored when [remote state](/docs/state/remote.html) is used.

* `-target=resource` - A [Resource
  Address](/docs/internals/resource-addressing.html) to target. This flag can
  be used multiple times. See below for more information.

* `-var 'foo=bar'` - Set a variable in the Terraform configuration. This flag
  can be set multiple times. Variable values are interpreted as
  [HCL](/docs/configuration/syntax.html#HCL), so list and map values can be
  specified via this flag.

* `-var-file=foo` - Set variables in the Terraform configuration from
  a [variable file](/docs/configuration/variables.html#variable-files). If
  a `terraform.tfvars` or any `.auto.tfvars` files are present in the current
  directory, they will be automatically loaded. `terraform.tfvars` is loaded
  first and the `.auto.tfvars` files after in alphabetical order. Any files
  specified by `-var-file` override any values set automatically from files in
  the working directory. This flag can be used multiple times.

## Resource Targeting

The `-target` option can be used to focus Terraform's attention on only a
subset of resources.
[Resource Address](/docs/internals/resource-addressing.html) syntax is used
to specify the constraint. The resource address is interpreted as follows:

* If the given address has a _resource spec_, only the specified resource
  is targeted. If the named resource uses `count` and no explicit index
  is specified in the address, all of the instances sharing the given
  resource name are targeted.

* The the given address _does not_ have a resource spec, and instead just
  specifies a module path, the target applies to all resources in the
  specified module _and_ all of the descendent modules of the specified
  module.

This targeting capability is provided for exceptional circumstances, such
as recovering from mistakes or working around Terraform limitations. It
is *not recommended* to use `-target` for routine operations, since this can
lead to undetected configuration drift and confusion about how the true state
of resources relates to configuration.

Instead of using `-target` as a means to operate on isolated portions of very
large configurations, prefer instead to break large configurations into
several smaller configurations that can each be independently applied.
[Data sources](/docs/configuration/data-sources.html) can be used to access
information about resources created in other configurations, allowing
a complex system architecture to be broken down into more managable parts
that can be updated independently.

## Security Warning

Saved plan files (with the `-out` flag) encode the configuration,
state, diff, and _variables_. Variables are often used to store secrets.
Therefore, the plan file can potentially store secrets.

Terraform itself does not encrypt the plan file. It is highly
recommended to encrypt the plan file if you intend to transfer it
or keep it at rest for an extended period of time.

Future versions of Terraform will make plan files more
secure.
