---
layout: "language"
page_title: "Workspaces via Terraform Cloud"
sidebar_current: "docs-workspaces"
---

# Workspaces via Terraform Cloud

Terraform Cloud expands the concept of workspaces to contain their own
execution context, stored values for input variables, state versioning, run
history, and more. Compared to workspaces provided in Terraform CLI by state
backends, they behave more like separate working directories. Rather than
being associated with one another by a single Terraform configuration,
Terraform Cloud workspaces are associated by a Terraform Cloud organization.
Each contains their own configuration and state. They can be integrated with
version control service providers to individually track separate
configurations in any number of VCS repositories or directories within those
repositories.

When used in Terraform CLI as part of Terraform Cloud's CLI-driven run
workflow, remote Terraform Cloud workspaces are mapped from a Terraform Cloud
organization to a single local configuration. This allows them to used in a
similar workflow to the one provided by remote state backends, and managed by
Terraform CLI via the same familiar `terraform workspace` subcommands (e.g.
`list`, `new`, `select`).

For more on configuring Terraform to use Terraform Cloud, see [TODO]().

For more on Terraform Cloud workspaces, see the [Terraform Cloud documentation
on workspaces](/docs/cloud/workspaces/index.html).
