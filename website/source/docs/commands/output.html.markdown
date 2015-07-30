---
layout: "docs"
page_title: "Command: output"
sidebar_current: "docs-commands-output"
description: |-
  The `terraform output` command is used to extract the value of an output variable from the state file.
---

# Command: output

The `terraform output` command is used to extract the value of
an output variable from the state file.

## Usage

Usage: `terraform output [options] NAME`

By default, `output` requires only a variable name and looks in the
current directory for the state file to query.

The command-line flags are all optional. The list of available flags are:

* `-state=path` - Path to the state file. Defaults to "terraform.tfstate".
* `-module=module_name` - The module path which has needed output.
    By default this is the root path. Other modules can be specified by
    a period-separated list. Example: "foo" would reference the module
    "foo" but "foo.bar" would reference the "bar" module in the "foo"
    module.
