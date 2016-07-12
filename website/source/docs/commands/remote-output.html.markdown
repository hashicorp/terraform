---
layout: "docs"
page_title: "Command: remote config"
sidebar_current: "docs-commands-remote-config"
description: |-
  The `terraform remote output` command is used to read an output
  variable from Terraform remote state. This command does not read
  or alter your existing configruation, and can be used without
  any remote state configured.
---

# Command: remote output

The `terraform remote output` command is used to read an output variable from
Terraform remote state. This command does not read or alter your existing
configruation, and can be used without any remote state configured.

## Usage

Usage: `terraform remote output [options] [NAME]`

Usage of the command is very similar to the 
[`terraform output`](/docs/commands/output.html) and
[`terraform remote config`](/docs/commands/remote-config.html) 
commands.

If `NAME` is supplied, only that output is returned.

The command-line flags are all optional. The list of available flags are:

* `-remote-backend=Atlas` - Specifies the type of remote backend. See
  `terraform remote config -help` for a list of supported backends. Defaults
  to `Atlas`.

* `-remote-config="k=v"` - Specifies configuration for the remote storage
  backend. This can be specified multiple times.

* `-no-color` - If specified, output won't contain any color.

* `-module=name` - If specified, returns the outputs for a specific module.
