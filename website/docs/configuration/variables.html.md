---
layout: "docs"
page_title: "Input Variables - Configuration Language"
sidebar_current: "docs-config-variables"
description: |-
  Input variables are parameters for Terraform modules.
  This page covers configuration syntax for variables.
---

# Input Variables

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Input Variables](../configuration-0-11/variables.html).

Input variables serve as parameters for a Terraform module, allowing aspects
of the module to be customized without altering the module's own source code,
and allowing modules to be shared between different configurations.

When you declare variables in the root module of your configuration, you can
set their values using CLI options and environment variables.
When you declare them in [child modules](./modules.html),
the calling module should pass values in the `module` block.

Input variable usage is introduced in the Getting Started guide section
[_Input Variables_](/intro/getting-started/variables.html).

-> **Note:** For brevity, input variables are often referred to as just
"variables" or "Terraform variables" when it is clear from context what sort of
variable is being discussed. Other kinds of variables in Terraform include
_environment variables_ (set by the shell where Terraform runs) and _expression
variables_ (used to indirectly represent a value in an
[expression](./expressions.html)).

## Declaring an Input Variable

Each input variable accepted by a module must be declared using a `variable`
block:

```hcl
variable "image_id" {
  type = string
}

variable "availability_zone_names" {
  type    = list(string)
  default = ["us-west-1a"]
}
```

The label after the `variable` keyword is a name for the variable, which must
be unique among all variables in the same module. This name is used to
assign a value to the variable from outside and to reference the variable's
value from within the module.

The name of a variable can be any valid [identifier](./syntax.html#identifiers)
_except_ the following:

- `source`
- `version`
- `providers`
- `count`
- `for_each`
- `lifecycle`
- `depends_on`
- `locals`

These names are reserved for meta-arguments in
[module configuration blocks](./modules.html), and cannot be
declared as variable names.

The variable declaration can optionally include a `type` argument to
specify what value types are accepted for the variable, as described
in the following section.

The variable declaration can also include a `default` argument. If present,
the variable is considered to be _optional_ and the default value will be used
if no value is set when calling the module or running Terraform. The `default`
argument requires a literal value and cannot reference other objects in the
configuration.

## Using Input Variable Values

Within the module that declared a variable, its value can be accessed from
within [expressions](./expressions.html) as `var.<NAME>`,
where `<NAME>` matches the label given in the declaration block:

```hcl
resource "aws_instance" "example" {
  instance_type = "t2.micro"
  ami           = var.image_id
}
```

The value assigned to a variable can be accessed only from expressions within
the module where it was declared.

## Type Constraints

The `type` argument in a `variable` block allows you to restrict the
[type of value](./expressions.html#types-and-values) that will be accepted as
the value for a variable. If no type constraint is set then a value of any type
is accepted.

While type constraints are optional, we recommend specifying them; they
serve as easy reminders for users of the module, and
allow Terraform to return a helpful error message if the wrong type is used.

Type constraints are created from a mixture of type keywords and type
constructors. The supported type keywords are:

* `string`
* `number`
* `bool`

The type constructors allow you to specify complex types such as
collections:

* `list(<TYPE>)`
* `set(<TYPE>)`
* `map(<TYPE>)`
* `object({<ATTR NAME> = <TYPE>, ... })`
* `tuple([<TYPE>, ...])`

The keyword `any` may be used to indicate that any type is acceptable. For
more information on the meaning and behavior of these different types, as well
as detailed information about automatic conversion of complex types, see
[Type Constraints](./types.html).

If both the `type` and `default` arguments are specified, the given default
value must be convertible to the specified type.

## Input Variable Documentation

Because the input variables of a module are part of its user interface, you can
briefly describe the purpose of each variable using the optional
`description` argument:

```hcl
variable "image_id" {
  type        = string
  description = "The id of the machine image (AMI) to use for the server."
}
```

The description should concisely explain the purpose
of the variable and what kind of value is expected. This description string
might be included in documentation about the module, and so it should be written
from the perspective of the user of the module rather than its maintainer. For
commentary for module maintainers, use comments.

## Assigning Values to Root Module Variables

When variables are declared in the root module of your configuration, they
can be set in a number of ways:

* [In a Terraform Enterprise workspace](/docs/enterprise/workspaces/variables.html).
* Individually, with the `-var` command line option.
* In variable definitions (`.tfvars`) files, either specified on the command line
  or automatically loaded.
* As environment variables.

The following sections describe these options in more detail. This section does
not apply to _child_ modules, where values for input variables are instead
assigned in the configuration of their parent module, as described in
[_Modules_](./modules.html).

### Variables on the Command Line

To specify individual modules on the command line, use the `-var` option
when running the `terraform plan` and `terraform apply` commands:

```
terraform apply -var="image_id=ami-abc123"
terraform apply -var='image_id_list=["ami-abc123","ami-def456"]'
terraform apply -var='image_id_map={"us-east-1":"ami-abc123","us-east-2":"ami-def456"}'
```

The `-var` option can be used any number of times in a single command.

### Variable Definitions (`.tfvars`) Files

To set lots of variables, it is more convenient to specify their values in
a _variable definitions file_ (with a filename ending in either `.tfvars`
or `.tfvars.json`) and then specify that file on the command line with
`-var-file`:

```
terraform apply -var-file="testing.tfvars"
```

-> **Note:** This is how Terraform Enterprise passes
[workspace variables](/docs/enterprise/workspaces/variables.html) to Terraform.

A variable definitions file uses the same basic syntax as Terraform language
files, but consists only of variable name assignments:

```hcl
image_id = "ami-abc123"
availability_zone_names = [
  "us-east-1a",
  "us-west-1c",
]
```

Terraform also automatically loads a number of variable definitions files
if they are present:

* Files named exactly `terraform.tfvars` or `terraform.tfvars.json`.
* Any files with names ending in `.auto.tfvars` or `.auto.tfvars.json`.

Files whose names end with `.json` are parsed instead as JSON objects, with
the root object properties corresponding to variable names:

```json
{
  "image_id": "ami-abc123",
  "availability_zone_names": ["us-west-1a", "us-west-1c"]
}
```

### Environment Variables

As a fallback for the other ways of defining variables, Terraform searches
the environment of its own process for environment variables named `TF_VAR_`
followed by the name of a declared variable.

This can be useful when running Terraform in automation, or when running a
sequence of Terraform commands in succession with the same variables.
For example, at a `bash` prompt on a Unix system:

```
$ export TF_VAR_image_id=ami-abc123
$ terraform plan
...
```

On operating systems where environment variable names are case-sensitive,
Terraform matches the variable name exactly as given in configuration, and
so the required environment variable name will usually have a mix of upper
and lower case letters as in the above example.

### Complex-typed Values

When variable values are provided in a variable definitions file, Terraform's
[usual syntax](./expressions.html#structural-types) can be used to assign
complex-typed values, like lists and maps.

Some special rules apply to the `-var` command line option and to environment
variables. For convenience, Terraform defaults to interpreting `-var` and
environment variable values as literal strings, which do not need to be quoted:

```
$ export TF_VAR_image_id=ami-abc123
```

However, if a root module variable uses a [type constraint](#type-constraints)
to require a complex value (list, set, map, object, or tuple), Terraform will
instead attempt to parse its value using the same syntax used within variable
definitions files, which requires cafeful attention to the string escaping rules
in your shell:

```
$ export TF_VAR_availability_zone_names='["us-west-1b","us-west-1d"]'
```

For readability, and to avoid the need to worry about shell escaping, we
recommend always setting complex variable values via varable definitions files.

### Variable Definition Precedence

The above mechanisms for setting variables can be used together in any
combination. If the same variable is assigned multiple values, Terraform uses
the _last_ value it finds, overriding any previous values.

Terraform loads variables in the following order, with later sources taking
precedence over earlier ones:

* Environment variables
* The `terraform.tfvars` file, if present.
* The `terraform.tfvars.json` file, if present.
* Any `*.auto.tfvars` or `*.auto.tfvars.json` files, processed in lexical order
  of their filenames.
* Any `-var` and `-var-file` options on the command line, in the order they
  are provided. (This includes variables set by a Terraform Enterprise
  workspace.)

~> **Important:** In Terraform 0.12 and later, variables with map and object
values behave the same way as other variables: the last value found overrides
the previous values. This is a change from previous versions of Terraform, which
would _merge_ map values instead of overriding them.
