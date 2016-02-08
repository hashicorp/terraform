---
layout: "docs"
page_title: "Command: validate"
sidebar_current: "docs-commands-validate"
description: |-
  The `terraform validate` command is used to validate the format and structure of the terraform files.
---

# Command: verify

The `terraform validate` command is used to validate the syntax of the terraform files.
Terraform performs a syntax check on all the terraform files in the directory, and will display an error if the file(s)
doesn't validate.

These errors include:

 * Interpolation in variable values, depends_on, module source etc.

 * Duplicate names in resource, modules and providers.

 * Missing variable values.

## Usage

Usage: `terraform validate [dir]`

By default, `validate` requires no flags and looks in the current directory
for the configurations.