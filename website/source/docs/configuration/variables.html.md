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

variable "zones" {
  default = ["us-east-1a", "us-east-1b"]
}
```

## Description

The `variable` block configures a single input variable for
a Terraform configuration. Multiple variables blocks can be used to
add multiple variables.

The `name` given to the variable block is the name used to
set the variable via the CLI as well as reference the variable
throughout the Terraform configuration.

Within the block (the `{ }`) is configuration for the variable.
These are the parameters that can be set:

  * `type` (optional) - If set this defines the type of the variable.
    Valid values are `string`, `list`, and `map`. If this field is omitted, the
    variable type will be inferred based on the `default`. If no `default` is
    provided, the type is assumed to be `string`.

  * `default` (optional) - This sets a default value for the variable.
    If no default is provided, the variable is considered required and
    Terraform will error if it is not set. The default value can be any of the
    data types Terraform supports. This is covered in more detail below.

  * `description` (optional) - A human-friendly description for
    the variable. This is primarily for documentation for users
    using your Terraform configuration. A future version of Terraform
    will expose these descriptions as part of some Terraform CLI
    command.

------

**Default values** can be strings, lists, or maps. If a default is specified,
it must match the declared type of the variable.

String values are simple and represent a basic key to value
mapping where the key is the variable name. An example is:

```
variable "key" {
  type    = "string"
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

A list can also be useful to store certain variables. For example:

```
variable "users" {
  type    = "list"
  default = ["admin", "ubuntu"]
}
```

The usage of maps, lists, strings, etc. is documented fully in the
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

[
  VALUE,
  ...
]

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
```

Maps and lists can be specified using environment variables as well using
[HCL](/docs/configuration/syntax.html#HCL) syntax in the value.

Given the variable declarations:

```
variable "somelist" {
  type = "list"
}
```

The variable could be set like so:

```
$ TF_VAR_somelist='["ami-abc123", "ami-bcd234"]' terraform plan
```

Similarly, for a map declared like:

```
variable "somemap" {
  type = "map"
}
```

The value can be set like this:

```
$ TF_VAR_somemap='{foo = "bar", baz = "qux"}' terraform plan
```

## Variable Files

<a id="variable-files"></a>

Variables can be collected in files and passed all at once using the
`-var-file=foo.tfvars` flag. 

If a "terraform.tfvars" file is present in the current directory, Terraform 
automatically loads it to populate variables. If the file is named something 
else, you can use the -var-file flag directly to specify a file. These files 
are the same syntax as Terraform configuration files. And like Terraform 
configuration files, these files can also be JSON.  The format for variables in 
`.tfvars` files is [HCL](/docs/configuration/syntax.html#HCL), with top level 
key/value pairs:

```
foo = "bar"
xyz = "abc"
somelist = [
  "one",
  "two",
]
somemap = {
  foo = "bar"
  bax = "qux"
}
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
