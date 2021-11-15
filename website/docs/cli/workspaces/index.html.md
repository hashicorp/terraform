---
layout: "docs"
page_title: "Managing Workspaces - Terraform CLI"
description: "Commands to list, select, create, and output workspaces. Workspaces help manage different groups of resources with one configuration."
---

# Managing Workspaces

In Terraform CLI, _workspaces_ are separate instances of
[state data](/docs/language/state/index.html) that can be used from the same working
directory. You can use workspaces to manage multiple non-overlapping groups of
resources with the same configuration.

- Every [initialized working directory](/docs/cli/init/index.html) has at least
  one workspace. (If you haven't created other workspaces, it is a workspace
  named `default`.)
- For a given working directory, only one workspace can be _selected_ at a time.
- Most Terraform commands (including [provisioning](/docs/cli/run/index.html)
  and [state manipulation](/docs/cli/state/index.html) commands) only interact
  with the currently selected workspace.
- Use [the `terraform workspace select` command](/docs/cli/commands/workspace/select.html)
  to change the currently selected workspace.
- Use the [`terraform workspace list`](/docs/cli/commands/workspace/list.html),
  [`terraform workspace new`](/docs/cli/commands/workspace/new.html), and
  [`terraform workspace delete`](/docs/cli/commands/workspace/delete.html) commands
  to manage the available workspaces in the current working directory.

The remainder of this section explains how to manage workspaces in Terraform CLI via the `workspaces` command. **For more information what workspaces are, their purpose, and how workspaces in Terraform CLI relate to Terraform Cloud workspaces, see the [Terraform Language documentation on Workspaces](/docs/language/workspaces/index.html).**
