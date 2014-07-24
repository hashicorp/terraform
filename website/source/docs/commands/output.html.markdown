---
layout: "docs"
page_title: "Command: output"
sidebar_current: "docs-commands-output"
---

# Command: output

The `terraform output` command is used to extract the value of
an output variable from the state file.

## Usage

Usage: `terraform output [options] NAME`

By default, `plan` requires only a variable name and looks in the
current directory for the state file to query.

The command-line flags are all optional. The list of available flags are:

* `-state=path` - Path to the state file. Defaults to "terraform.tfstate".

