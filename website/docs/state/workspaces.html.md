---
layout: "docs"
page_title: "State: Workspaces"
sidebar_current: "docs-state-workspaces"
description: |-
  Workspaces allow the use of multiple states with a single configuration directory.
---

# Workspaces

A _workspace_ is a named container for Terraform state. With multiple
workspaces, a single directory of Terraform configuration can be used to
manage multiple distinct sets of infrastructure resources.

Terraform state determines what resources it manages based on what
exists in the state. This is how `terraform plan` determines what isn't
created, what needs to be updated, etc. The full details of state can be
found on [the _purpose_ page](/docs/state/purpose.html).

Multiple workspaces are currently supported by the following backends:

 * [AzureRM](/docs/backends/types/azurerm.html)
 * [Consul](/docs/backends/types/consul.html)
 * [S3](/docs/backends/types/s3.html)

In the 0.9 line of Terraform releases, this concept was known as "environment".
It was renamed in 0.10 based on feedback about confusion caused by the
overloading of the word "environment" both within Terraform itself and within
organizations that use Terraform.

## Using Workspaces

Terraform starts with a single workspace named "default". This
workspace is special both because it is the default and also because
it cannot ever be deleted. If you've never explicitly used workspaces, then
you've only ever worked on the "default" workspace.

Workspaces are managed with the `terraform workspace` set of commands. To
create a new workspace and switch to it, you can use `terraform workspace new`;
to switch environments you can use `terraform workspace select`; etc.

For example, creating a new workspace:

```text
$ terraform workspace new bar
Created and switched to workspace "bar"!

You're now on a new, empty workspace. Workspaces isolate their state,
so if you run "terraform plan" Terraform will not see any existing state
for this configuration.
```

As the command says, if you run `terraform plan`, Terraform will not see
any existing resources that existed on the default (or any other) workspace.
**These resources still physically exist,** but are managed in another
Terraform workspace.

## Current Workspace Interpolation

Within your Terraform configuration, you may include the name of the current
workspace using the `${terraform.workspace}` interpolation sequence. This can
be used anywhere interpolations are allowed.

Referencing the current workspace is useful for changing behavior based
on the workspace. For example, for non-default workspaces, it may be useful
to spin up smaller cluster sizes. For example:

```hcl
resource "aws_instance" "example" {
  count = "${terraform.workspace == "default" ? 5 : 1}"

  # ... other arguments
}
```

Another popular use case is using the workspace name as part of naming or
tagging behavior:

```hcl
resource "aws_instance" "example" {
  tags {
    Name = "web - ${terraform.workspace}"
  }

  # ... other arguments
}
```

## Best Practices

Workspaces can be used to manage small differences between development,
staging, and production, but they **should not** be treated as the only
isolation mechanism. As Terraform configurations get larger, it's much more
manageable and safer to split one large configuration into many
smaller ones linked together with the `terraform_remote_state` data source.
This allows teams to delegate ownership and reduce the potential impact of
changes. For *each* smaller configuration, you can use workspaces to model
the differences between development, staging, and production. However, if you
have one large Terraform configuration, it is riskier and not recommended to
use workspaces to handle those differences.

[The `terraform_remote_state` data source](/docs/providers/terraform/d/remote_state.html)
accepts a `workspace` name to target. Therefore, you can link
together multiple independently managed Terraform configurations with the same
environment easily, with each configuration itself having multiple workspaces.

While workspaces are available to all,
[Terraform Enterprise](https://www.hashicorp.com/products/terraform/)
provides an interface and API for managing sets of configurations linked
with `terraform_remote_state` and viewing them all as a single environment.

Workspaces alone are useful for isolating a set of resources to test
changes during development. For example, it is common to associate a
branch in a VCS with a temporary workspace so new features can be developed
without affecting the default workspace.

Future Terraform versions and workspace enhancements will enable
Terraform to track VCS branches with a workspace to help verify only certain
branches can make changes to a Terraform workspace.

## Workspace Internals

Workspaces are technically equivalent to renaming your state file. They
aren't any more complex than that. Terraform wraps this simple notion with
a set of protections and support for remote state.

For local state, Terraform stores the workspace states in a directory called
`terraform.tfstate.d`. This directory should be be treated similarly to
local-only `terraform.tfstate`); some teams commit these files to version
control, although using a remote backend instead is recommended when there are
multiple collaborators.

For [remote state](/docs/state/remote.html), the workspaces are stored
directly in the configured [backend](/docs/backends). For example, if you
use [Consul](/docs/backends/types/consul.html), the workspaces are stored
by appending the environment name to the state path. To ensure that
workspace names are stored correctly and safely in all backends, the name
must be valid to use in a URL path segment without escaping.

The important thing about workspace internals is that workspaces are
meant to be a shared resource. They aren't a private, local-only notion
(unless you're using purely local state and not committing it).

The "current workspace" name is stored only locally in the ignored
`.terraform` directory. This allows multiple team members to work on
different workspaces concurrently.
