---
layout: "commands-workspace"
page_title: "Command: workspace select"
sidebar_current: "docs-workspace-sub-select"
description: |-
  The terraform workspace select command is used to choose a workspace.
---

# Command: workspace select

The `terraform workspace select` command is used to choose a different
workspace to use for further operations.

## Usage

Usage: `terraform workspace select [NAME]`

This command will select another workspace. The named workspace must already
exist.

## Example

```
$ terraform workspace list
  default
* development
  jsmith-test

$ terraform workspace select default
Switched to workspace "default".
```
