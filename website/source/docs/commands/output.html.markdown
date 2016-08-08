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

Usage: `terraform output [options] [NAME]`

With no additional arguments, `output` will display all the outputs for the root module.
If an output `NAME` is specified, only the value of that output is printed.

The command-line flags are all optional. The list of available flags are:

* `-json` - If specified, the outputs are formatted as a JSON object, with
    a key per output. If `NAME` is specified, only the output specified will be
    returned. This can be piped into tools such as `jq` for further processing.
* `-state=path` - Path to the state file. Defaults to "terraform.tfstate".
  Ignored when [remote state](/docs/state/remote/index.html) is used.
* `-module=module_name` - The module path which has needed output.
    By default this is the root path. Other modules can be specified by
    a period-separated list. Example: "foo" would reference the module
    "foo" but "foo.bar" would reference the "bar" module in the "foo"
    module.
