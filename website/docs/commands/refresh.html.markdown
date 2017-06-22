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

Usage: `terraform refresh [options] [dir]`

By default, `refresh` requires no flags and looks in the current directory
for the configuration and state file to refresh.

The command-line flags are all optional. The list of available flags are:

* `-backup=path` - Path to the backup file. Defaults to `-state-out` with
  the ".backup" extension. Disabled by setting to "-".

* `-input=true` - Ask for input for variables if not directly set.

* `-lock=true` - Lock the state file when locking is supported.

* `-lock-timeout=0s` - Duration to retry a state lock.

* `-no-color` - If specified, output won't contain any color.

* `-state=path` - Path to read and write the state file to. Defaults to "terraform.tfstate".
  Ignored when [remote state](/docs/state/remote.html) is used.

* `-state-out=path` - Path to write updated state file. By default, the
  `-state` path will be used. Ignored when
  [remote state](/docs/state/remote.html) is used.

* `-target=resource` - A [Resource
  Address](/docs/internals/resource-addressing.html) to target. Operation will
  be limited to this resource and its dependencies. This flag can be used
  multiple times.

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
