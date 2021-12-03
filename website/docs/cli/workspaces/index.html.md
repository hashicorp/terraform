---
layout: "docs"
page_title: "Managing Workspaces - Terraform CLI"
description: "Commands to list, select, create, and output workspaces. Workspaces help manage different groups of resources with one configuration."
---

# Managing Workspaces

A _workspace_ is a long-lived execution context that Terraform uses to manage a
particular group of resources.

In Terraform CLI, workspaces are implemented as separate instances of
[state data](/docs/language/state/index.html) that can be used from the same working
directory. You can use workspaces to manage multiple non-overlapping groups of
resources with the same configuration.

-> **Note:** Workspaces are implemented differently in Terraform Cloud, since
Terraform Cloud has a global view of your infrastructure instead of a single
working directory. For details, see
[Terraform Cloud: Workspaces](/docs/cloud/workspaces/index.html).

## Basics of Workspaces

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
  to manage workspaces in the current working directory.

## Collaborating With Workspaces

Terraform CLI supports using multiple workspaces with:

- Local state
- [Terraform Cloud](/docs/language/settings/terraform-cloud.html)
  (when using tags to select multiple workspaces)
- _Most_ [remote state backends](/docs/language/settings/backends/index.html)

When using remote state, the set of available workspaces is shared, but the name
of the currently selected workspace is only saved locally. So everyone sees the
same workspaces available, but can work independently on a given workspace.

## When to Use or Not Use Workspaces

Using a single configuration with multiple workspaces is a good fit when you
need several _similar and equal_ instances of the same infrastructure pattern.

In some cases, this is a good fit for multiple deployment environments (dev,
prod, etc.). However, it's also common to want your deployment environments to
be _less similar and less equal_ — for example, production might have tighter
controls on access to Terraform state and a more complex set of resources. In
these cases, using a single configuration plus workspaces will become very
awkward, and we don't recommend it. Instead, we recommend refactoring that
infrastructure pattern into a collection of
[reusable modules](/docs/language/modules/develop/index.html) and calling those
modules from several small configurations.

## Workspace Internals

- For local state, Terraform stores the default workspace's state in the
  `terraform.tfstate` file, and all other workspace states in a directory called
  `terraform.tfstate.d`.
- For remote backends, the backend determines how workspace states are stored.
  For example, [Consul](/docs/language/settings/backends/consul.html)
  appends each workspace's name to the configured state path.
