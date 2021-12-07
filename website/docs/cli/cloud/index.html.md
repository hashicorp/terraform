---
layout: "docs"
page_title: "Using Terraform Cloud - Terraform CLI"
---

# Using Terraform Cloud with Terraform

The Terraform Cloud CLI Integration allows you to use Terraform Cloud and Terraform Enterprise on the command line. In the documentation Terraform Cloud instructions also apply to Terraform Enterprise, except where explicitly stated.

Using Terraform Cloud through the command line is called the [CLI-driven run workflow](/docs/cloud/run/cli.html). When you use the CLI workflow, operations like `terraform plan` or `terraform apply` are remotely executed in Terraform Cloud's run environment by default, with log output streaming to the local terminal. This lets you use Terraform Cloud features within the familiar Terraform CLI workflow, including variables encrypted at rest in a Terraform Cloud workspace, cost estimates, and policy checking.

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

- [Terraform Cloud Settings](/docs/cli/cloud/settings.html) documents the `cloud` block that you must add to your configuration to enable Terraform Cloud support.
- [Initializing and Migrating](/docs/cli/cloud/migrating.html) describes
how to start using Terraform Cloud with a working directory that already has state data.
- [Command Line Arguments](/docs/cli/cloud/command-line-arguments.html) lists the Terraform command flags that are specific to using Terraform with Terraform Cloud.

Refer to the [CLI-driven Run Workflow](/docs/cloud/run/cli.html) for more details about how to use Terraform Cloud from the command line.
