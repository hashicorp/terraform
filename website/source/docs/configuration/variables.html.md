---
layout: "docs"
page_title: "Configuring Variables"
sidebar_current: "docs-config-variables"
description: |-
  Variables define the parameterization of Terraform configurations. Variables can be overridden via the CLI. Variable usage is covered in more detail in the getting started guide. This page covers configuration syntax for variables.
---

# Variable Configuration

Variables define the parameterization of Terraform configurations.
Variables can be overridden via the CLI. Variable usage is
covered in more detail in the
[getting started guide](/intro/getting-started/variables.html).
This page covers configuration syntax for variables.

This page assumes you're familiar with the
[configuration syntax](/docs/configuration/syntax.html)
already.

## Example

A variable configuration looks like the following:

```
variable "key" {
    type = "string"
}

variable "images" {
    type = "map"

	default = {
		us-east-1 = "image-1234"
		us-west-2 = "image-4567"
	}
}
```

## Description

The `variable`  block configures a single input variable for
a Terraform configuration. Multiple variables blocks can be used to
add multiple variables.

The `name` given to the variable block is the name used to
set the variable via the CLI as well as reference the variable
throughout the Terraform configuration.

Within the block (the `{ }`) is configuration for the variable.
These are the parameters that can be set:

  * `type` (optional) - If set this defines the type of the variable.
    Valid values are `string` and `map`. In older versions of Terraform
    this parameter did not exist, and the type was inferred from the
    default value, defaulting to `string` if no default was set. If a
    type is not specified, the previous behavior is maintained. It is
    recommended to set variable types explicitly in preference to relying
    on inferrence - this allows variables of type `map` to be set in the
    `terraform.tfvars` file without requiring a default value to be set.

  * `default` (optional) - If set, this sets a default value
    for the variable. If this isn't set, the variable is required
    and Terraform will error if not set. The default value can be
    a string or a mapping. This is covered in more detail below.

  * `description` (optional) - A human-friendly description for
    the variable. This is primarily for documentation for users
    using your Terraform configuration. A future version of Terraform
    will expose these descriptions as part of some Terraform CLI
    command.

------

**Default values** can be either strings or maps, and if specified
must match the declared type of the variable. If no value is supplied
for a variable of type `map`, the values must be supplied in a
`terraform.tfvars` file - they cannot be input via the console.

String values are simple and represent a basic key to value
mapping where the key is the variable name. An example is:

```
variable "key" {
    type = "string"

	default = "value"
}
```

A map allows a key to contain a lookup table. This is useful
for some values that change depending on some external pivot.
A common use case for this is mapping cloud images to regions.
An example:

```
variable "images" {
    type = "map"

	default = {
		us-east-1 = "image-1234"
		us-west-2 = "image-4567"
	}
}
```

The usage of maps, strings, etc. is documented fully in the
[interpolation syntax](/docs/configuration/interpolation.html)
page.

## Syntax

The full syntax is:

```
variable NAME {
	[type = TYPE]
	[default = DEFAULT]
	[description = DESCRIPTION]
}
```

where `DEFAULT` is:

```
VALUE

{
	KEY = VALUE
	...
}
```

## Environment Variables

Environment variables can be used to set the value of a variable.
The key of the environment variable must be `TF_VAR_name` and the value
is the value of the variable.

For example, given the configuration below:

```
variable "image" {}
```

The variable can be set via an environment variable:

```
$ TF_VAR_image=foo terraform apply
...
```

## Variable Files

Variables can be collected in files and passed all at once using the 
`-var-file=foo.tfvars` flag. The format for variables in `.tfvars`
files is:
```
foo = "bar"
xyz = "abc"

```

The flag can be used multiple times per command invocation:

```
terraform apply -var-file=foo.tfvars -var-file=bar.tfvars
```

**Note** If a variable is defined in more than one file passed, the last 
variable file (reading left to right) will be the definition used. Put more 
simply, the last time a variable is defined is the one which will be used.

### Precedence example:

Both these files have the variable `baz` defined:

_foo.tfvars_
```
baz = "foo"
```

_bar.tfvars_
```
baz = "bar"
```

When they are passed in the following order:

```
terraform apply -var-file=foo.tfvars -var-file=bar.tfvars
```

The result will be that `baz` will contain the value `bar` because `bar.tfvars`
has the last definition loaded.


