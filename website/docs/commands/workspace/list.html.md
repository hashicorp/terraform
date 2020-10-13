---
layout: "commands-workspace"
page_title: "Command: workspace list"
sidebar_current: "docs-commands-workspace-sub-list"
description: |-
  The terraform workspace list command is used to list all existing workspaces.
---

# Command: workspace list

The `terraform workspace list` command is used to list all existing workspaces.

## Usage

Usage: `terraform workspace list`

The command will list all existing workspaces. The current workspace is
indicated using an asterisk (`*`) marker.

## Example

```
$ terraform workspace list
  default
* development
  jsmith-test
```
