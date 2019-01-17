---
layout: "docs"
page_title: "Input Variables - 0.11 Configuration Language"
sidebar_current: "docs-conf-old-variables"
description: |-
  Input variables are parameters for Terraform modules.
  This page covers configuration syntax for variables.
---

# Input Variables

-> **Note:** This page is about Terraform 0.11 and earlier. For Terraform 0.12
and later, see
[Configuration Language: Input Variables](../configuration/variables.html).

Input variables serve as parameters for a Terraform module.

When used in the root module of a configuration, variables can be set from CLI
arguments and environment variables. For [_child_ modules](./modules.html),
they allow values to pass from parent to child.

Input variable usage is introduced in the Getting Started guide section
[_Input Variables_](/intro/getting-started/variables.html).

This page assumes you're familiar with the
[configuration syntax](./syntax.html)
already.

## Example

Input variables can be defined as follows:

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

The `variable` block configures a single input variable for a Terraform module.
Each block declares a single variable.

The name given in the block header is used to assign a value to the variable
via the CLI and to reference the variable elsewhere in the configuration.

Within the block body (between `{ }`) is configuration for the variable,
which accepts the following arguments:

- `type` (Optional) - If set this defines the type of the variable. Valid values
  are `string`, `list`, and `map`. If this field is omitted, the variable type
  will be inferred based on `default`. If no `default` is provided, the type
  is assumed to be `string`.

- `default` (Optional) - This sets a default value for the variable. If no
  default is provided, Terraform will raise an error if a value is not provided
  by the caller. The default value can be of any of the supported data types,
  as described below. If `type` is also set, the given value must be
  of the specified type.

- `description` (Optional) - A human-friendly description for the variable. This
  is primarily for documentation for users using your Terraform configuration.
  When a module is published in [Terraform Registry](https://registry.terraform.io/),
  the given description is shown as part of the documentation.

The name of a variable can be any valid identifier. However, due to the
interpretation of [module configuration blocks](./modules.html),
the names `source`, `version` and `providers` are reserved for Terraform's own
use and are thus not recommended for any module intended to be used as a
child module.

The default value of an input variable must be a _literal_ value, containing
no interpolation expressions. To assign a name to an expression so that it
may be re-used within a module, use [Local Values](./locals.html)
instead.

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

Terraform performs automatic conversion from string values to numeric and
boolean values based on context, so in practice string variables may be used
to set arguments of any primitive type. For boolean values in particular
there are some caveats, described under [_Booleans_](#booleans) below.

### Maps

A map value is a lookup table from string keys to string values. This is
useful for selecting a value based on some other provided value.

A common use of maps is to create a table of machine images per region,
as follows:

```hcl
variable "images" {
  type    = "map"
  default = {
    "us-east-1" = "image-1234"
    "us-west-2" = "image-4567"
  }
}
```

### Lists

A list value is an ordered sequence of strings indexed by integers starting
with zero. For example:

```hcl
variable "users" {
  type    = "list"
  default = ["admin", "ubuntu"]
}
```

### Booleans

Although Terraform can automatically convert between boolean and string
values, there are some subtle implications of these conversions that should
be completely understood when using boolean values with input variables.

It is recommended for now to specify boolean values for variables as the
strings `"true"` and `"false"`, to avoid some caveats in the conversion
process. A future version of Terraform will properly support boolean values
and so relying on the current behavior could result in
backwards-incompatibilities at that time.

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

- "1" and "true" become `true`
- "0" and "false" become `false`

The behavior of conversion in _this_ direction (string to boolean) will _not_
change in future Terraform versions. Therefore, using these string values
rather than literal booleans is recommended when using input variables.

## Environment Variables

Environment variables can be used to set the value of an input variable in
the root module. The name of the environment variable must be
`TF_VAR_` followed by the variable name, and the value is the value of the
variable.

For example, given the configuration below:

```hcl
variable "image" {}
```

The variable can be set via an environment variable:

```shell
$ TF_VAR_image=foo terraform apply
```

Maps and lists can be specified using environment variables as well using
[HCL](./syntax.html#HCL) syntax in the value.

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

Values for the input variables of a root module can be gathered in
_variable definition files_ and passed together using the `-var-file=FILE`
option.

For all files which match `terraform.tfvars` or `*.auto.tfvars` present in the
current directory, Terraform automatically loads them to populate variables. If
the file is located somewhere else, you can pass the path to the file using the
`-var-file` flag. It is recommended to name such files with names ending in
`.tfvars`.

Variables files use HCL or JSON syntax to define variable values. Strings, lists
or maps may be set in the same manner as the default value in a `variable` block
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
specified on the command line. If a particular variable is defined in more than
one variable file, the last value specified is effective.

### Variable Merging

When multiple values are provided for the same input variable, map values are
merged while all other values are overriden by the last definition.

For example, if you define a variable twice on the command line:

```shell
$ terraform apply -var foo=bar -var foo=baz
```

Then the value of `foo` will be `baz`, since it was the last definition seen.

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

Definition files passed using the `-var-file` flag will always be evaluated after
those in the working directory.

Values passed within definition files or with `-var` will take precedence over
`TF_VAR_` environment variables, as environment variables are considered defaults.
