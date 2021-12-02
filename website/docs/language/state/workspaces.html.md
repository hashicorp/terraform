---
layout: "language"
page_title: "Workspaces"
description: |-
  A workspace is a long-lived execution context that Terraform uses to manage a
  particular group of resources.
---

# Workspaces

A workspace is a long-lived execution context that Terraform uses to manage a
particular group of resources.

Since Terraform CLI and Terraform Cloud have different execution models, they
represent workspaces differently. But in both cases, the point of a workspace is
to use a Terraform configuration to manage a particular group of real-world
resources over some period of time.

## Workspaces in Terraform CLI

Terraform CLI implements workspaces as separate state files.

This is because Terraform CLI's primary execution context is just a working
directory, and the working directory already contains most of what Terraform
needs to manage resources (a Terraform configuration, possibly one or more
`.tfvars` files to set variable values, cached provider binaries, etc.). A
richer definition of workspaces wouldn't make any sense on the CLI, because
things like the configuration to use, the Terraform version to execute, etc. are
already predetermined before Terraform tries to do anything.

Thus, Terraform CLI's workspaces are just a convenient way to manage several
distinct instances of infrastructure from a single working directory. A single
Terraform configuration can be re-used to manage multiple groups of similar
resources (for example, separate dev and prod deployment environments), and
using multiple workspaces allows you to do this without duplicating the working
directory for each instance.

By default, Terraform CLI uses a single workspace named `default`.

For more information about creating and selecting workspaces in Terraform CLI,
see [CLI: Managing Workspaces](/docs/cli/workspaces/index.html).

## Workspaces in Terraform Cloud

Terraform Cloud implements workspaces as an entire Terraform execution context.
In addition to maintaining state data for a particular group of resources, each
workspace can specify which Terraform version it expects to run, has some way to
obtain a Terraform configuration (either from version control, from a user
running Terraform CLI with Terraform Cloud integration enabled, or uploaded via
the API), and specifies values for Terraform variables and environment
variables.

This is because Terraform Cloud has a global view of Terraform-managed
infrastructure across your organization, instead of running from an arbitrary
local directory like Terraform CLI.

A Terraform Cloud workspace is essentially a separate instance of state for a
configuration, _and_ a dedicated working directory, _and_ a dedicated shell
environment, _and_ some additional information like run history.

For more information, see
[Terraform Cloud: Workspaces](/docs/cloud/workspaces/index.html).

## Connecting Terraform CLI to Terraform Cloud Workspaces

You can integrate Terraform CLI with Terraform Cloud by specifying one or more
workspaces in a [`cloud` block](/docs/language/settings/terraform-cloud.html).
This lets you use familiar commands to perform remote runs in Terraform Cloud,
as well as perform some state manipulation actions that don't have direct
equivalents in Terraform Cloud's interface.

Cloud integration is scoped to a particular Terraform configuration. You can
connect a working directory to a single Terraform Cloud workspace, or to
multiple workspaces that share a set of tags; in either case, it only makes
sense to connect to workspaces that use the same configuration as the working
directory in question.

## Referencing the Current Workspace Name

Within a Terraform configuration, the expression `terraform.workspace` evaluates
to the name of the current workspace.

Referencing the current workspace is useful for changing behavior based
on the workspace. For example, for non-production workspaces, it may be useful
to spin up smaller cluster sizes:

```hcl
resource "aws_instance" "example" {
  count = "${terraform.workspace == "app-prod" ? 5 : 1}"

  # ... other arguments
}
```

Another popular use case is using the workspace name as part of naming or
tagging behavior:

```hcl
resource "aws_instance" "example" {
  tags = {
    Name = "web - ${terraform.workspace}"
  }

  # ... other arguments
}
```

