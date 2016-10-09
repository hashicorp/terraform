---
layout: "docs"
page_title: "Command: import"
sidebar_current: "docs-commands-import"
description: |-
  The `terraform import` command is used to import existing resources into Terraform.
---

# Command: import

The `terraform import` command is used to
[import existing resources](/docs/import/index.html)
into Terraform.

## Usage

Usage: `terraform import [options] ADDRESS ID`

Import will find the existing resource from ID and import it into your Terraform
state at the given ADDRESS.

ADDRESS must be a valid [resource address](/docs/internals/resource-addressing.html).
Because any resource address is valid, the import command can import resources
into modules as well directly into the root of your state.

ID is dependent on the resource type being imported. For example, for AWS
instances it is the instance ID (`i-abcd1234`) but for AWS Route53 zones
it is the zone ID (`Z12ABC4UGMOZ2N`). Please reference the provider documentation for details
on the ID format. If you're unsure, feel free to just try an ID. If the ID
is invalid, you'll just receive an error message.

The command-line flags are all optional. The list of available flags are:

* `-backup=path` - Path to backup the existing state file. Defaults to
  the `-state-out` path with the ".backup" extension. Set to "-" to disable
  backups.

* `-input=true` - Whether to ask for input for provider configuration.

* `-state=path` - The path to read and save state files (unless state-out is
  specified). Ignored when [remote state](/docs/state/remote/index.html) is used.

* `-state-out=path` - Path to write the final state file. By default, this is
  the state path. Ignored when [remote state](/docs/state/remote/index.html) is
  used.

## Provider Configuration

To access the provider that the resource is being imported from, Terraform
will ask you for access credentials. If you don't want to be asked for input,
verify that all environment variables for your provider are set.

The import command cannot read provider configuration from a Terraform
configuration file.

## Example: AWS Instance

This example will import an AWS instance:

```
$ terraform import aws_instance.foo i-abcd1234
```

## Example: Import to Module

The example below will import an AWS instance into a module:

```
$ terraform import module.foo.aws_instance.bar i-abcd1234
```
