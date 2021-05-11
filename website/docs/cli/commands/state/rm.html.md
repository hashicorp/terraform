---
layout: "docs"
page_title: "Command: state rm"
sidebar_current: "docs-commands-state-sub-rm"
description: |-
  The `terraform state rm` command removes items from the Terraform state.
---

# Command: state rm

The `terraform state rm` command is used to remove items from the
[Terraform state](/docs/language/state/index.html). This command can remove
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
in [resource addressing format](/docs/cli/state/resource-addressing.html).

This command doesn't normally accept any command line options, except in
the special situations described in the following paragraphs.

For configurations using
[the `remote` backend](/docs/language/settings/backends/remote.html)
only, `terraform state rm`
also accepts the option
[`-ignore-remote-version`](/docs/language/settings/backends/remote.html#command-line-arguments).

For configurations using
[the `local` state rm](/docs/language/settings/backends/local.html) only,
`terraform state rm` also accepts the legacy options
[`-state`, `-state-out`, and `-backup`](/docs/language/settings/backends/local.html#command-line-arguments).

## Example: Remove a Resource

The example below removes the `packet_device` resource named `worker`:

```shell
$ terraform state rm 'packet_device.worker'
```

## Example: Remove a Module

The example below removes the entire module named `foo`:

```shell
$ terraform state rm 'module.foo'
```

## Example: Remove a Module Resource

The example below removes the `packet_device` resource named `worker` inside a module named `foo`:

```shell
$ terraform state rm 'module.foo.packet_device.worker'
```

## Example: Remove a Resource configured with count

The example below removes the first instance of a `packet_device` resource named `worker` configured with
[`count`](/docs/language/meta-arguments/count.html):

```shell
$ terraform state rm 'packet_device.worker[0]'
```

## Example: Remove a Resource configured with for_each

The example below removes the `"example"` instance of a `packet_device` resource named `worker` configured with
[`for_each`](/docs/language/meta-arguments/for_each.html):

Linux, Mac OS, and UNIX:

```shell
$ terraform state rm 'packet_device.worker["example"]'
```

PowerShell:

```shell
$ terraform state rm 'packet_device.worker[\"example\"]'
```

Windows `cmd.exe`:

```shell
$ terraform state rm packet_device.worker[\"example\"]
```
