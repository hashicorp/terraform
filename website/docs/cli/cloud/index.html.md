---
layout: "docs"
page_title: "Using Terraform Cloud - Terraform CLI"
---

# Using Terraform Cloud with Terraform

The Terraform Cloud CLI Integration allows you to use Terraform Cloud and Terraform Enterprise on the command line. In the documentation Terraform Cloud instructions also apply to Terraform Enterprise, except where explicitly stated.

Operations like `terraform plan` or `terraform apply` are remotely executed in Terraform Cloud's run
environment, with log output streaming to the local terminal. This allows usage of Terraform Cloud's
features — for example, using variables encrypted at rest in a Terraform Cloud workspace, cost
estimates, and policy checking — within the familiar Terraform CLI workflow.

Workspaces can also be configured for local execution, in which case only state is stored in
Terraform Cloud. In this mode, Terraform Cloud behaves just like a standard state backend.

-> **Note:** The Cloud integration for Terraform was added in Terraform 1.1.0; for previous
versions, see the [remote backend documentation](/docs/language/settings/backends/remote.html). See
also: [Migrating from the remote
backend](/docs/cli/cloud/migrating.html)

-> **Note:** This integration supports Terraform Enterprise as well. Throughout all the
documentation, the platform will be referred to as Terraform Cloud, with any Terraform
Enterprise-specific details explicitly stated. The minimum required version of Terraform Enterprise
is 202201-1.

## Documentation Summary

* [Terraform Cloud Settings](/docs/cli/cloud/settings.html) documents the form of the `cloud` block that enables Terraform Cloud support for a Terraform configuration.
* [Initializing and Migrating](/docs/cli/cloud/migrating.html) describes
how to start using Terraform Cloud with a working directory that already has state data.
* [Command Line Arguments](/docs/cli/cloud/command-line-arguments.html) lists the Terraform command flags that are specific to using Terraform with Terraform Cloud.

For more information, see Terraform Cloud's documentation on [Terraform Runs and Remote
Operations](/docs/cloud/run/index.html).
