---
layout: "docs"
page_title: "Managing Workspaces - Terraform CLI"
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

-> **Note:** Terraform Cloud and Terraform CLI both have features called
"workspaces," but they're slightly different. Terraform Cloud's workspaces
behave more like completely separate working directories.

## The Purpose of Workspaces

Since most of the resources you can manage with Terraform don't include a unique
name as part of their configuration, it's common to use the same Terraform
configuration to provision multiple groups of similar resources.

Terraform relies on [state](/docs/language/state/index.html) to associate resources with
real-world objects, so if you run the same configuration multiple times with
completely separate state data, Terraform can manage many non-overlapping groups
of resources. In some cases you'll want to change
[variable values](/docs/language/values/variables.html) for these different
resource collections (like when specifying differences between staging and
production deployments), and in other cases you might just want many instances
of a particular infrastructure pattern.

The simplest way to maintain multiple instances of a configuration with
completely separate state data is to use multiple
[working directories](/docs/cli/init/index.html) (with different
[backend](/docs/language/settings/backends/configuration.html) configurations per directory, if you
aren't using the default `local` backend).

However, this isn't always the most _convenient_ way to handle separate states.
Terraform installs a separate cache of plugins and modules for each working
directory, so maintaining multiple directories can waste bandwidth and disk
space. You must also update your configuration code from version control
separately for each directory, reinitialize each directory separately when
changing the configuration, etc.

Workspaces allow you to use the same working copy of your configuration and the
same plugin and module caches, while still keeping separate states for each
collection of resources you manage.

## Interactions with Terraform Cloud Workspaces

Terraform Cloud organizes infrastructure using workspaces, but its workspaces
act more like completely separate working directories; each Terraform Cloud
workspace has its own Terraform configuration, set of variable values, state
data, run history, and settings.

These two kinds of workspaces are different, but related. When using Terraform
CLI as a frontend for Terraform Cloud, you associate the current working
directory with one or more remote workspaces by configuring
[the `remote` backend](/docs/language/settings/backends/remote.html). If you associate the
directory with multiple workspaces (using a name prefix), you can use the
`terraform workspace` commands to select which remote workspace to use.

For more information about using Terraform CLI with Terraform Cloud, see
[CLI-driven Runs](/docs/cloud/run/cli.html) in the Terraform Cloud docs.
