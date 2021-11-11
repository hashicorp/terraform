---
layout: "language"
page_title: "Workspaces"
sidebar_current: "docs-workspaces"
description: |-
  A workspace is an isolated instance of a Terraform configuration with its own state. They typically model multiple deployment environments within a single working directory in Terraform CLI, and are the main organizational unit of infrastructure provisioned within a Terraform Cloud organization.
---

# Workspaces

A workspace is an isolated instance of a Terraform configuration with its own state. In Terraform
CLI, workspaces typically model multiple _deployment environments_ (e.g. `production`, `staging`,
etc) within a single working directory, allowing a single configuration to be provisioned multiple
times. In Terraform Cloud, they are the main organizational unit for all Terraform configurations
provisioned within a Terraform Cloud organization.

* By default - and when using a configured state backend - workspaces are technically equivalent to
renaming your state file. They aren't any more complex than that. Terraform wraps this simple notion
with a set of protections and support for remote state. [Workspaces via Terraform state
backends](/docs/language/workspaces/via-backends.html) describes using this type of workspace.

* Using Terraform Cloud, workspaces are more flexible in that they are not associated with a
particular configuration but by a Terraform Cloud organization. Two related workspaces (representing
the `prod` and `staging` deployment environments of a single set of resources) may _or may not_ be
associated with the same configuration or state. In this sense, a Terraform Cloud workspace behaves
more like a completely separate working directory, and is typically named by both the set of
resources it contains as well as the deployment environment it provisions to (e.g.
`networking-prod-us-east`, `networking-staging-us-east`, etc). Terraform Cloud workspaces can also
contain their own execution context, stored values for input variables, state
versioning, run history, and more. [Workspaces via Terraform
Cloud](/docs/language/workspaces/via-terraform-cloud.html) describes using Terraform
Cloud workspaces from Terraform itself.

## Using Workspaces

Terraform starts with a single workspace named "default". This
workspace is special both because it is the default and also because
it cannot ever be deleted. If you've never explicitly used workspaces, then
you've only ever worked on the "default" workspace.

Workspaces are managed with the `terraform workspace` set of commands. To
create a new workspace and switch to it, you can use `terraform workspace new`;
to switch workspaces you can use `terraform workspace select`; etc.

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

For more on how to manage workspaces in Terraform CLI, see [Managing
Workspaces](/docs/cli/workspaces/index.html).

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
  tags = {
    Name = "web - ${terraform.workspace}"
  }

  # ... other arguments
}
```

