---
layout: "docs"
page_title: "Command: refresh"
sidebar_current: "docs-commands-refresh"
description: |-
  The `terraform refresh` command is used to reconcile the state Terraform knows about (via its state file) with the real-world infrastructure. This can be used to detect any drift from the last-known state, and to update the state file.
---

# Command: refresh

The `terraform refresh` command is used to reconcile the state Terraform
knows about (via its state file) with the real-world infrastructure.
This can be used to detect any drift from the last-known state, and to
update the state file.

This does not modify infrastructure, but does modify the state file.
If the state is changed, this may cause changes to occur during the next
plan or apply.

## Usage

Usage: `terraform refresh [options]`

The `terraform refresh` command accepts the following options:

* `-compact-warnings` - If Terraform produces any warnings that are not
  accompanied by errors, show them in a more compact form that includes only
  the summary messages.

* `-input=true` - Ask for input for variables if not directly set.

* `-lock=true` - Lock the state file when locking is supported.

* `-lock-timeout=0s` - Duration to retry a state lock.

* `-no-color` - If specified, output won't contain any color.

* `-parallelism=n` - Limit the number of concurrent operation as Terraform
  [walks the graph](/docs/internals/graph.html#walking-the-graph). Defaults
  to 10.

* `-target=resource` - A [Resource
  Address](/docs/cli/state/resource-addressing.html) to target. Operation will
  be limited to this resource and its dependencies. This flag can be used
  multiple times.

* `-var 'foo=bar'` - Set a variable in the Terraform configuration. This flag
  can be set multiple times. Variable values are interpreted as
  [literal expressions](/docs/language/expressions/types.html) in the
  Terraform language, so list and map values can be specified via this flag.

* `-var-file=foo` - Set variables in the Terraform configuration from
  a [variable file](/docs/language/values/variables.html#variable-definitions-tfvars-files). If
  a `terraform.tfvars` or any `.auto.tfvars` files are present in the current
  directory, they will be automatically loaded. `terraform.tfvars` is loaded
  first and the `.auto.tfvars` files after in alphabetical order. Any files
  specified by `-var-file` override any values set automatically from files in
  the working directory. This flag can be used multiple times.

For configurations using
[the `local` backend](/docs/language/settings/backends/local.html) only,
`terraform refresh` also accepts the legacy options
[`-state`, `-state-out`, and `-backup`](/docs/language/settings/backends/local.html#command-line-arguments).
