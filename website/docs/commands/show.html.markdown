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

* `-no-color` - Disables output with coloring

