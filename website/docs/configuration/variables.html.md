---
layout: "language"
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

> **Hands-on:** Try the [Customize Terraform Configuration with Variables](https://learn.hashicorp.com/tutorials/terraform/variables?in=terraform/configuration-language&utm_source=WEBSITE&utm_medium=WEB_IO&utm_offer=ARTICLE_PAGE&utm_content=DOCS) tutorial on HashiCorp Learn.

Input variables serve as parameters for a Terraform module, allowing aspects
of the module to be customized without altering the module's own source code,
and allowing modules to be shared between different configurations.

When you declare variables in the root module of your configuration, you can
set their values using CLI options and environment variables.
When you declare them in [child modules](/docs/configuration/blocks/modules/index.html),
the calling module should pass values in the `module` block.

If you're familiar with traditional programming languages, it can be useful to
compare Terraform modules to function definitions:

- Input variables are like function arguments.
- [Output values](./outputs.html) are like function return values.
- [Local values](./locals.html) are like a function's temporary local variables.

-> **Note:** For brevity, input variables are often referred to as just
"variables" or "Terraform variables" when it is clear from context what sort of
variable is being discussed. Other kinds of variables in Terraform include
_environment variables_ (set by the shell where Terraform runs) and _expression
variables_ (used to indirectly represent a value in an
[expression](/docs/configuration/expressions/index.html)).

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

variable "docker_ports" {
  type = list(object({
    internal = number
    external = number
    protocol = string
  }))
  default = [
    {
      internal = 8300
      external = 8300
      protocol = "tcp"
    }
  ]
}
```

The label after the `variable` keyword is a name for the variable, which must
be unique among all variables in the same module. This name is used to
assign a value to the variable from outside and to reference the variable's
value from within the module.

The name of a variable can be any valid [identifier](./syntax.html#identifiers)
_except_ the following: `source`, `version`, `providers`, `count`, `for_each`, `lifecycle`, `depends_on`, `locals`.

These names are reserved for meta-arguments in
[module configuration blocks](/docs/configuration/blocks/modules/syntax.html), and cannot be
declared as variable names.

## Arguments

Terraform CLI defines the following optional arguments for variable declarations:

- [`default`][inpage-default] - A default value which then makes the variable optional.
- [`type`][inpage-type] - This argument specifies what value types are accepted for the variable.
- [`description`][inpage-description] - This specifies the input variable's documentation.
- [`validation`][inpage-validation] - A block to define validation rules, usually in addition to type constraints.
- [`sensitive`][inpage-sensitive] - Limits Terraform UI output when the variable is used in configuration.

### Default values

[inpage-default]: #default-values

The variable declaration can also include a `default` argument. If present,
the variable is considered to be _optional_ and the default value will be used
if no value is set when calling the module or running Terraform. The `default`
argument requires a literal value and cannot reference other objects in the
configuration.

### Type Constraints

[inpage-type]: #type-constraints

The `type` argument in a `variable` block allows you to restrict the
[type of value](/docs/configuration/expressions/types.html) that will be accepted as
the value for a variable. If no type constraint is set then a value of any type
is accepted.

While type constraints are optional, we recommend specifying them; they
can serve as helpful reminders for users of the module, and they
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

### Input Variable Documentation

[inpage-description]: #input-variable-documentation

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

### Custom Validation Rules

[inpage-validation]: #custom-validation-rules

-> This feature was introduced in Terraform CLI v0.13.0.

In addition to Type Constraints as described above, a module author can specify
arbitrary custom validation rules for a particular variable using a `validation`
block nested within the corresponding `variable` block:

```hcl
variable "image_id" {
  type        = string
  description = "The id of the machine image (AMI) to use for the server."

  validation {
    condition     = length(var.image_id) > 4 && substr(var.image_id, 0, 4) == "ami-"
    error_message = "The image_id value must be a valid AMI id, starting with \"ami-\"."
  }
}
```

The `condition` argument is an expression that must use the value of the
variable to return `true` if the value is valid, or `false` if it is invalid.
The expression can refer only to the variable that the condition applies to,
and _must not_ produce errors.

If the failure of an expression is the basis of the validation decision, use
[the `can` function](./functions/can.html) to detect such errors. For example:

```hcl
variable "image_id" {
  type        = string
  description = "The id of the machine image (AMI) to use for the server."

  validation {
    # regex(...) fails if it cannot find a match
    condition     = can(regex("^ami-", var.image_id))
    error_message = "The image_id value must be a valid AMI id, starting with \"ami-\"."
  }
}
```

If `condition` evaluates to `false`, Terraform will produce an error message
that includes the sentences given in `error_message`. The error message string
should be at least one full sentence explaining the constraint that failed,
using a sentence structure similar to the above examples.

### Suppressing Values in CLI Output

[inpage-sensitive]: #suppressing-values-in-cli-output

-> This feature was introduced in Terraform CLI v0.14.0.

Setting a variable as `sensitive` prevents Terraform from showing its value in the `plan` or `apply` output, when that variable is used within a configuration.

Sensitive values are still recorded in the [state](/docs/state/index.html), and so will be visible to anyone who is able to access the state data. For more information, see [_Sensitive Data in State_](/docs/state/sensitive-data.html).

A provider can define [an attribute as sensitive](/docs/extend/best-practices/sensitive-state.html#using-the-sensitive-flag), which prevents the value of that attribute from being displayed in logs or regular output. The `sensitive` argument on variables allows users to replicate this behavior for values in their configuration, by defining a variable as `sensitive`.

Define a variable as sensitive by setting the `sensitive` argument to `true`:

```
variable "user_information" {
  type = object({
    name    = string
    address = string
  })
  sensitive = true
}

resource "some_resource" "a" {
  name    = var.user_information.name
  address = var.user_information.address
}
```

Using this variable throughout your configuration will obfuscate the value from display in `plan` or `apply` output:

```
Terraform will perform the following actions:

  # some_resource.a will be created
  + resource "some_resource" "a" {
      + name    = (sensitive)
      + address = (sensitive)
    }

Plan: 1 to add, 0 to change, 0 to destroy.
```

In some cases where a sensitive variable is used in a nested block, the whole block can be redacted. This happens with resources which can have multiple blocks of the same type, where the values must be unique. This looks like:

```
# main.tf

resource "some_resource" "a" {
  nested_block {
    user_information  = var.user_information # a sensitive variable
    other_information = "not sensitive data"
  }
}

# CLI output

Terraform will perform the following actions:

  # some_resource.a will be updated in-place
  ~ resource "some_resource" "a" {
      ~ nested_block {
          # At least one attribute in this block is (or was) sensitive,
          # so its contents will not be displayed.
        }
    }

```

#### Cases where Terraform may disclose a sensitive variable

A `sensitive` variable is a configuration-centered concept, and values are sent to providers without any obfuscation. A provider error could disclose a value if that value is included in the error message. For example, a provider might return the following error even if "foo" is a sensitive value: `"Invalid value 'foo' for field"`

If a resource attribute is used as, or part of, the provider-defined resource id, an `apply` will disclose the value. In the example below, the `prefix` attribute has been set to a sensitive variable, but then that value ("jae") is later disclosed as part of the resource id:

```
  # random_pet.animal will be created
  + resource "random_pet" "animal" {
      + id        = (known after apply)
      + length    = 2
      + prefix    = (sensitive)
      + separator = "-"
    }

Plan: 1 to add, 0 to change, 0 to destroy.

...

random_pet.animal: Creating...
random_pet.animal: Creation complete after 0s [id=jae-known-mongoose]
```

## Using Input Variable Values

Within the module that declared a variable, its value can be accessed from
within [expressions](/docs/configuration/expressions/index.html) as `var.<NAME>`,
where `<NAME>` matches the label given in the declaration block:

-> **Note:** Input variables are _created_ by a `variable` block, but you
_reference_ them as attributes on an object named `var`.

```hcl
resource "aws_instance" "example" {
  instance_type = "t2.micro"
  ami           = var.image_id
}
```

The value assigned to a variable can only be accessed in expressions within
the module where it was declared.

## Assigning Values to Root Module Variables

When variables are declared in the root module of your configuration, they
can be set in a number of ways:

* [In a Terraform Cloud workspace](/docs/cloud/workspaces/variables.html).
* Individually, with the `-var` command line option.
* In variable definitions (`.tfvars`) files, either specified on the command line
  or automatically loaded.
* As environment variables.

The following sections describe these options in more detail. This section does
not apply to _child_ modules, where values for input variables are instead
assigned in the configuration of their parent module, as described in
[_Modules_](/docs/configuration/blocks/modules/index.html).

### Variables on the Command Line

To specify individual variables on the command line, use the `-var` option
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

-> **Note:** This is how Terraform Cloud passes
[workspace variables](/docs/cloud/workspaces/variables.html) to Terraform.

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

When variable values are provided in a variable definitions file, you can use
Terraform's usual syntax for
[literal expressions](/docs/configuration/expressions/types.html#literal-expressions)
to assign complex-typed values, like lists and maps.

Some special rules apply to the `-var` command line option and to environment
variables. For convenience, Terraform defaults to interpreting `-var` and
environment variable values as literal strings, which do not need to be quoted:

```
$ export TF_VAR_image_id=ami-abc123
```

However, if a root module variable uses a [type constraint](#type-constraints)
to require a complex value (list, set, map, object, or tuple), Terraform will
instead attempt to parse its value using the same syntax used within variable
definitions files, which requires careful attention to the string escaping rules
in your shell:

```
$ export TF_VAR_availability_zone_names='["us-west-1b","us-west-1d"]'
```

For readability, and to avoid the need to worry about shell escaping, we
recommend always setting complex variable values via variable definitions files.

### Variable Definition Precedence

The above mechanisms for setting variables can be used together in any
combination. If the same variable is assigned multiple values, Terraform uses
the _last_ value it finds, overriding any previous values. Note that the same
variable cannot be assigned multiple values within a single source.

Terraform loads variables in the following order, with later sources taking
precedence over earlier ones:

* Environment variables
* The `terraform.tfvars` file, if present.
* The `terraform.tfvars.json` file, if present.
* Any `*.auto.tfvars` or `*.auto.tfvars.json` files, processed in lexical order
  of their filenames.
* Any `-var` and `-var-file` options on the command line, in the order they
  are provided. (This includes variables set by a Terraform Cloud
  workspace.)

~> **Important:** In Terraform 0.12 and later, variables with map and object
values behave the same way as other variables: the last value found overrides
the previous values. This is a change from previous versions of Terraform, which
would _merge_ map values instead of overriding them.
