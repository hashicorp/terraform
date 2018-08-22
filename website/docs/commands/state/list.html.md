---
layout: "commands-state"
page_title: "Command: state list"
sidebar_current: "docs-state-sub-list"
description: |-
  The terraform state list command is used to list resources within a Terraform state.
---

# Command: state list

The `terraform state list` command is used to list resources within a
[Terraform state](/docs/state/index.html).

## Usage

Usage: `terraform state list [options] [address...]`

The command will list all resources in the state file matching the given
addresses (if any). If no addresses are given, all resources are listed.

The resources listed are sorted according to module depth order followed
by alphabetical. This means that resources that are in your immediate
configuration are listed first, and resources that are more deeply nested
within modules are listed last.

For complex infrastructures, the state can contain thousands of resources.
To filter these, provide one or more patterns to the command. Patterns are
in [resource addressing format](/docs/commands/state/addressing.html).

The command-line flags are all optional. The list of available flags are:

* `-state=path` - Path to the state file. Defaults to "terraform.tfstate".
  Ignored when [remote state](/docs/state/remote.html) is used.
* `-id=id` - ID of resources to show. Ignored when unset.

## Example: All Resources

This example will list all resources, including modules:

```
$ terraform state list
aws_instance.foo
aws_instance.bar[0]
aws_instance.bar[1]
module.elb.aws_elb.main
```

## Example: Filtering by Resource

This example will only list resources for the given name:

```
$ terraform state list aws_instance.bar
aws_instance.bar[0]
aws_instance.bar[1]
```

## Example: Filtering by Module

This example will only list resources in the given module:

```
$ terraform state list module.elb
module.elb.aws_elb.main
```

## Example: Filtering by ID

This example will only list the resource whose ID is specified on the
command line. This is useful to find where in your configuration a
specific resource is located.

```
$ terraform state list -id=sg-1234abcd
module.elb.aws_security_group.sg
```
