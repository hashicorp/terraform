---
layout: "docs"
page_title: "Terraform Cloud Overview"
sidebar_current: "configuring-terraform-cloud"
description: "Configure Terraform to use Terraform Cloud"
---

# Using Terraform Cloud with Terraform

Terraform Cloud can be used on the command line with Terraform itself. Using Terraform Cloud in this
way is referred to as the [CLI-driven run workflow](/docs/cloud/run/cli.html).

Operations like `terraform plan` or `terraform apply` are remotely executed in Terraform Cloud's run
environment, with log output streaming to the local terminal. This allows usage of Terraform Cloud's
features - for example, using variables encrypted at rest in a Terraform Cloud workspace, cost
estimates, and policy checking - into the familiar Terraform CLI workflow.

Workspaces can also be configured for local execution, in which case only state is stored in
Terraform Cloud. In this mode, Terraform Cloud behaves just like a standard state backend.

-> **Note:** The Cloud integration for Terraform was added in Terraform 1.1.0; for previous
versions, see the [remote backend documentation](/docs/language/settings/backends/remote.html). See
also: [Migrating from the remote
backend](/docs/cli/configuring-terraform-cloud/migrating-from-the-remote-backend.html)

-> **Note:** This integration supports Terraform Enterprise as well. Throughout all the
documentation, the platform will be referred to as Terraform Cloud, with any Terraform
Enterprise-specific details explicitly stated. The minimum required version of Terraform Enterprise
is 202201-1.

### Documentation Summary

* [Initialization with the `cloud` block](/docs/cli/configuring-terraform-cloud/initialization.html) documents the form of the `terraform` settings block used to initialize Terraform Cloud for a Terraform configuration.
* [Command Line Arguments](/docs/cli/configuring-terraform-cloud/command-line-arguments.html) lists any Terraform command flags that are specific to using Terraform with Terraform Cloud.
* [Migrating from the remote
backend](/docs/cli/configuring-terraform-cloud/migrating-from-the-remote-backend.html) describes
how to migrate from using the `remote` backend - Terraform Cloud's previous implementation of the
CLI-driven run workflow - to this native integration.

For more information, see Terraform Cloud's documentation on [Terraform Runs and Remote
Operations](/docs/cloud/run/index.html).
