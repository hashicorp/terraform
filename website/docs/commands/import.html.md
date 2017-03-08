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

* `-config=path` - Path to directory of Terraform configuration files that
  configure the provider for import. This defaults to your working directory.
  If this directory contains no Terraform configuration files, the provider
  must be configured via manual input or environmental variables.

* `-input=true` - Whether to ask for input for provider configuration.

* `-lock=true` - Lock the state file when locking is supported.

* `-lock-timeout=0s` - Duration to retry a state lock.

* `-no-color` - If specified, output won't contain any color.

* `-provider=provider` - Specified provider to use for import. The value should be a provider
  alias in the form `TYPE.ALIAS`, such as "aws.eu". This defaults to the normal
  provider based on the prefix of the resource being imported. You usually
  don't need to specify this.

* `-state=path` - The path to read and save state files (unless state-out is
  specified). Ignored when [remote state](/docs/state/remote.html) is used.

* `-state-out=path` - Path to write the final state file. By default, this is
  the state path. Ignored when [remote state](/docs/state/remote.html) is
  used.

* `-var 'foo=bar'` - Set a variable in the Terraform configuration. This flag
  can be set multiple times. Variable values are interpreted as
  [HCL](/docs/configuration/syntax.html#HCL), so list and map values can be
  specified via this flag. This is only useful with the `-config` flag.

* `-var-file=foo` - Set variables in the Terraform configuration from
   a [variable file](/docs/configuration/variables.html#variable-files). If
  any file matching "*.tfvars" are present, they will be automatically loaded
  in alphabetical order. Any files specified by `-var-file` override any values
  set automatically from files in the working directory. This flag can be used
  multiple times. This is only useful with the `-config` flag.

## Provider Configuration

Terraform will attempt to load configuration files that configure the
provider being used for import. If no configuration files are present or
no configuration for that specific provider is present, Terraform will
prompt you for access credentials. You may also specify environmental variables
to configure the provider.

The only limitation Terraform has when reading the configuration files
is that the import provider configurations must not depend on non-variable
inputs. For example, a provider configuration cannot depend on a data
source.

As a working example, if you're importing AWS resources and you have a
configuration file with the contents below, then Terraform will configure
the AWS provider with this file.

```hcl
variable "access_key" {}
variable "secret_key" {}

provider "aws" {
  access_key = "${var.access_key}"
  secret_key = "${var.secret_key}"
}
```

You can force Terraform to explicitly not load your configuration by
specifying `-config=""` (empty string). This is useful in situations where
you want to manually configure the provider because your configuration
may not be valid.

## Example: AWS Instance

This example will import an AWS instance:

```shell
$ terraform import aws_instance.foo i-abcd1234
```

## Example: Import to Module

The example below will import an AWS instance into a module:

```shell
$ terraform import module.foo.aws_instance.bar i-abcd1234
```
