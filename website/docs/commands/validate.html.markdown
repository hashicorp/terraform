---
layout: "docs"
page_title: "Command: validate"
sidebar_current: "docs-commands-validate"
description: |-
  The `terraform validate` command is used to validate the syntax of the terraform files.
---

# Command: validate

The `terraform validate` command validates the configuration files in a
directory, referring only to the configuration and not accessing any remote
services such as remote state, provider APIs, etc.

Validate runs checks that verify whether a configuration is syntactically
valid and internally consistent, regardless of any provided variables or
existing state. It is thus primarily useful for general verification of
reusable modules, including correctness of attribute names and value types.

It is safe to run this command automatically, for example as a post-save
check in a text editor or as a test step for a re-usable module in a CI
system.

Validation requires an initialized working directory with any referenced
plugins and modules installed. To initialize a working directory for
validation without accessing any configured remote backend, use:

```
$ terraform init -backend=false
```

If dir is not specified, then the current directory will be used.

To verify configuration in the context of a particular run (a particular
target workspace, input variable values, etc), use the `terraform plan`
command instead, which includes an implied validation check.

## Usage

Usage: `terraform validate [options] [dir]`

By default, `validate` requires no flags and looks in the current directory
for the configurations.

The command-line flags are all optional. The available flags are:

- `-json` - Produce output in a machine-readable JSON format, suitable for
  use in text editor integrations and other automated systems. Always disables
  color.

- `-no-color` - If specified, output won't contain any color.
