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

```hcl
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

- `type` (optional) - If set this defines the type of the variable. Valid values
  are `string`, `list`, and `map`. If this field is omitted, the variable type
  will be inferred based on the `default`. If no `default` is provided, the type
  is assumed to be `string`.

- `default` (optional) - This sets a default value for the variable. If no
  default is provided, the variable is considered required and Terraform will
  error if it is not set. The default value can be any of the data types
  Terraform supports. This is covered in more detail below.

- `description` (optional) - A human-friendly description for the variable. This
  is primarily for documentation for users using your Terraform configuration. A
  future version of Terraform will expose these descriptions as part of some
  Terraform CLI command.

-> **Note**: Default values can be strings, lists, or maps. If a default is
specified, it must match the declared type of the variable.

### Strings

String values are simple and represent a basic key to value
mapping where the key is the variable name. An example is:

```hcl
variable "key" {
  type    = "string"
  default = "value"
}
```

A multi-line string value can be provided using heredoc syntax.

```hcl
variable "long_key" {
  type = "string"
  default = <<EOF
This is a long key.
Running over several lines.
EOF
}
```

### Maps

A map allows a key to contain a lookup table. This is useful
for some values that change depending on some external pivot.
A common use case for this is mapping cloud images to regions.
An example:

```hcl
variable "images" {
  type = "map"
  default = {
    us-east-1 = "image-1234"
    us-west-2 = "image-4567"
  }
}
```

### Lists

A list can also be useful to store certain variables. For example:

```hcl
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

```text
variable NAME {
  [type = TYPE]
  [default = DEFAULT]
  [description = DESCRIPTION]
}
```

where `DEFAULT` is:

```text
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

### Booleans

Although it appears Terraform supports boolean types, they are instead
silently converted to string types. The implications of this are subtle and
should be completely understood if you plan on using boolean values.

It is instead recommended you avoid using boolean values for now and use
explicit strings. A future version of Terraform will properly support booleans
and using the current behavior could result in backwards-incompatibilities in
the future.

For a configuration such as the following:

```hcl
variable "active" {
  default = false
}
```

The false is converted to a string `"0"` when running Terraform.

Then, depending on where you specify overrides, the behavior can differ:

- Variables with boolean values in a `tfvars` file will likewise be converted to
  "0" and "1" values.

- Variables specified via the `-var` command line flag will be literal strings
  "true" and "false", so care should be taken to explicitly use "0" or "1".

- Variables specified with the `TF_VAR_` environment variables will be literal
  string values, just like `-var`.

A future version of Terraform will fully support first-class boolean
types which will make the behavior of booleans consistent as you would
expect. This may break some of the above behavior.

When passing boolean-like variables as parameters to resource configurations
that expect boolean values, they are converted consistently:

- "1", "true", "t" all become `true`
- "0", "false", "f" all become `false`

The behavior of conversion above will likely not change in future
Terraform versions. Therefore, simply using string values rather than
booleans for variables is recommended.

## Environment Variables

Environment variables can be used to set the value of a variable.
The key of the environment variable must be `TF_VAR_name` and the value
is the value of the variable.

For example, given the configuration below:

```hcl
variable "image" {}
```

The variable can be set via an environment variable:

```shell
$ TF_VAR_image=foo terraform apply
```

Maps and lists can be specified using environment variables as well using
[HCL](/docs/configuration/syntax.html#HCL) syntax in the value.

For a list variable like so:

```hcl
variable "somelist" {
  type = "list"
}
```

The variable could be set like so:

```shell
$ TF_VAR_somelist='["ami-abc123", "ami-bcd234"]' terraform plan
```

Similarly, for a map declared like:

```hcl
variable "somemap" {
  type = "map"
}
```

The value can be set like this:

```shell
$ TF_VAR_somemap='{foo = "bar", baz = "qux"}' terraform plan
```

## Variable Files

Variables can be collected in files and passed all at once using the
`-var-file=foo.tfvars` flag.

For all files which match `terraform.tfvars` or `*.auto.tfvars` present in the
current directory, Terraform automatically loads them to populate variables. If
the file is located somewhere else, you can pass the path to the file using the
`-var-file` flag.

Variables files use HCL or JSON to define variable values. Strings, lists or
maps may be set in the same manner as the default value in a `variable` block
in Terraform configuration. For example:

```hcl
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

The `-var-file` flag can be used multiple times per command invocation:

```shell
$ terraform apply -var-file=foo.tfvars -var-file=bar.tfvars
```

-> **Note**: Variable files are evaluated in the order in which they are
specified on the command line. If a variable is defined in more than one
variable file, the last value specified is effective.

### Variable Merging

When variables are conflicting, map values are merged and all other values are
overridden. Map values are always merged.

For example, if you set a variable twice on the command line:

```shell
$ terraform apply -var foo=bar -var foo=baz
```

Then the value of `foo` will be `baz` since it was the last value seen.

However, for maps, the values are merged:

```shell
$ terraform apply -var 'foo={quux="bar"}' -var 'foo={bar="baz"}'
```

The resulting value of `foo` will be:

```shell
{
  quux = "bar"
  bar = "baz"
}
```

There is no way currently to unset map values in Terraform. Whenever a map
is modified either via variable input or being passed into a module, the
values are always merged.

### Variable Precedence

Both these files have the variable `baz` defined:

_foo.tfvars_

```hcl
baz = "foo"
```

_bar.tfvars_

```hcl
baz = "bar"
```

When they are passed in the following order:

```shell
$ terraform apply -var-file=foo.tfvars -var-file=bar.tfvars
```

The result will be that `baz` will contain the value `bar` because `bar.tfvars`
has the last definition loaded.

Definitions passed using the `-var-file` flag will always be evaluated after
those in the working directory.
