---
layout: "docs"
page_title: "Command: show"
sidebar_current: "docs-commands-show"
description: |-
  The `terraform show` command is used to provide human-readable output from a state or plan file. This can be used to inspect a plan to ensure that the planned operations are expected, or to inspect the current state as Terraform sees it.
---

# Command: show

The `terraform show` command is used to provide human-readable output
from a state or plan file. This can be used to inspect a plan to ensure
that the planned operations are expected, or to inspect the current state
as Terraform sees it.

## Usage

Usage: `terraform show [options] [path]`

You may use `show` with a path to either a Terraform state file or plan
file. If no path is specified, the current state will be shown.

The command-line flags are all optional. The list of available flags are:

* `-format=name` - Produces output in an alternative format. Currently "json"
  is the only supported alternative format. See below for more information
  and caveats regarding the JSON output.

* `-module-depth=n` - Specifies the depth of modules to show in the output.
  By default this is -1, which will expand all.

* `-no-color` - Disables output with coloring

## JSON Output

As a convenience for intepreting Terraform data using external tools, Terraform
can produce detailed plan and state information in JSON format.

However, since Terraform is still quickly evolving we are unable to guarantee
100% compatibility with the current JSON data structures in future versions,
and so the current data structures are not documented in detail.

Please use this feature sparingly and with care, and be ready to update any
integrations when moving to newer versions of Terraform.

~> **Warning** The JSON output is generally more detailed than the
human-readable output, and in particular can include *sensitive information*.
The JSON output must therefore be stored and transmitted with care.
