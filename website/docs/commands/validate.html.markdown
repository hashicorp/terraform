---
layout: "docs"
page_title: "Command: validate"
sidebar_current: "docs-commands-validate"
description: |-
  The `terraform validate` command is used to validate the syntax of the terraform files.
---

# Command: validate

The `terraform validate` command is used to validate the syntax of the terraform files.
Terraform performs a syntax check on all the terraform files in the directory,
and will display an error if any of the files doesn't validate.

This command **does not** check formatting (e.g. tabs vs spaces, newlines, comments etc.).

The following can be reported:

 * invalid [HCL](https://github.com/hashicorp/hcl) syntax (e.g. missing trailing quote or equal sign)
 * invalid HCL references (e.g. variable name or attribute which doesn't exist)
 * same `provider` declared multiple times
 * same `module` declared multiple times
 * same `resource` declared multiple times
 * invalid `module` name
 * interpolation used in places where it's unsupported
 	(e.g. `variable`, `depends_on`, `module.source`, `provider`)
 * missing value for a variable (none of `-var foo=...` flag,
   `-var-file=foo.vars` flag, `TF_VAR_foo` environment variable,
   `terraform.tfvars`, or default value in the configuration)

## Usage

Usage: `terraform validate [options] [dir]`

By default, `validate` requires no flags and looks in the current directory
for the configurations.

The command-line flags are all optional. The available flags are:

* `-no-color` - Disables output with coloring.

* `-var 'foo=bar'` - Set a variable in the Terraform configuration. This flag
  can be set multiple times. Variable values are interpreted as
  [HCL](/docs/configuration/syntax.html#HCL), so list and map values can be
  specified via this flag.

* `-var-file=foo` - Set variables in the Terraform configuration from
   a [variable file](/docs/configuration/variables.html#variable-files). If
  "terraform.tfvars" is present, it will be automatically loaded first. Any
  files specified by `-var-file` override any values in a "terraform.tfvars".
  This flag can be used multiple times.
