---
layout: "commands-state"
page_title: "Command: state rm"
sidebar_current: "docs-state-sub-rm"
description: |-
  The `terraform state rm` command removes items from the Terraform state.
---

# Command: state rm

The `terraform state rm` command is used to remove items from the
[Terraform state](/docs/state/index.html). This command can remove
single resources, single instances of a resource, entire modules,
and more.

## Usage

Usage: `terraform state rm [options] ADDRESS...`

Remove one or more items from the Terraform state.

Items removed from the Terraform state are _not physically destroyed_.
Items removed from the Terraform state are only no longer managed by
Terraform. For example, if you remove an AWS instance from the state, the AWS
instance will continue running, but `terraform plan` will no longer see that
instance.

There are various use cases for removing items from a Terraform state
file. The most common is refactoring a configuration to no longer manage
that resource (perhaps moving it to another Terraform configuration/state).

The state will only be saved on successful removal of all addresses.
If any specific address errors for any reason (such as a syntax error),
the state will not be modified at all.

This command will output a backup copy of the state prior to saving any
changes. The backup cannot be disabled. Due to the destructive nature
of this command, backups are required.

This command requires one or more addresses that point to a resources in the
state. Addresses are
in [resource addressing format](/docs/commands/state/addressing.html).

The command-line flags are all optional. The list of available flags are:

* `-backup=path` - Path where Terraform should write the backup state. This
  can't be disabled. If not set, Terraform will write it to the same path as
  the statefile with a backup extension.

* `-state=path` - Path to a Terraform state file to use to look up
  Terraform-managed resources. By default it will use the configured backend,
  or the default "terraform.tfstate" if it exists.

## Example: Remove a Resource

The example below removes a single resource in a module:

```
$ terraform state rm module.foo.packet_device.worker[0]
```

## Example: Remove a Module

The example below removes an entire module:

```
$ terraform state rm module.foo
```
