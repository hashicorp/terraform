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
variable "key" {}

variable "images" {
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

The `NAME` given to the variable block is the name used to
set the variable via the CLI as well as reference the variable
throughout the Terraform configuration.

Within the block (the `{ }`) is configuration for the variable.
These are the parameters that can be set:

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

**Default values** can be either strings or maps. If a default
value is omitted and the variable is required, the value assigned
via the CLI must be a string.

String values are simple and represent a basic key to value
mapping where the key is the variable name. An example is:

```
variable "key" {
	default = "value"
}
```

A map allows a key to contain a lookup table. This is useful
for some values that change depending on some external pivot.
A common use case for this is mapping cloud images to regions.
An example:

```
variable "images" {
	default = {
		us-east-1 = "image-1234"
		us-west-2 = "image-4567"
	}
}
```

The usage of maps, strings, etc. is documented fully in the
[interpolation syntax](/docs/configuration/interpolation.html)
page.

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

## Syntax

The full syntax is:

```
variable NAME {
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
