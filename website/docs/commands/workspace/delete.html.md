---
layout: "commands-workspace"
page_title: "Command: workspace delete"
sidebar_current: "docs-workspace-sub-delete"
description: |-
  The terraform workspace delete command is used to delete a workspace.
---

# Command: workspace delete

The `terraform workspace delete` command is used to delete an existing workspace.

## Usage

Usage: `terraform workspace delete [NAME]`

This command will delete the specified workspace.

To delete an workspace, it must already exist, it must have an empty state,
and it must not be your current workspace. If the workspace state is not empty,
Terraform will not allow you to delete it unless the `-force` flag is specified.

If you delete a workspace with a non-empty state (via `-force`), then resources
may become "dangling". These are resources that physically exist but that
Terraform can no longer manage. This is sometimes preferred: you want
Terraform to stop managing resources so they can be managed some other way.
Most of the time, however, this is not intended and so Terraform protects you
from getting into this situation.

The command-line flags are all optional. The only supported flag is:

* `-force` - Delete the workspace even if its state is not empty. Defaults to false.

## Example

```
$ terraform workspace delete example
Deleted workspace "example".
```
