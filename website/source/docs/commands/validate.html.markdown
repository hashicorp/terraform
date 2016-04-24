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

## Usage

Usage: `terraform validate [dir]`

By default, `validate` requires no flags and looks in the current directory
for the configurations.