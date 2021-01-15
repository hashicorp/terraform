---
layout: "language"
page_title: "Provider: Terraform"
sidebar_current: "docs-terraform-index"
description: |-
  The special `terraform_remote_state` data source is used to access outputs from shared infrastructure.
---

# The Built-In `terraform` Provider

Terraform includes one built-in data source:
[`terraform_remote_state`](/docs/language/state/remote-state-data.html), which
provides access to root module outputs from some other Terraform configuration.

This data source is implemented by a built-in provider, whose
[source address](/docs/language/providers/requirements.html#source-addresses)
is `terraform.io/builtin/terraform`. You do not need to require or configure
this provider in order to use the `terraform_remote_state` data source; it is
always available.

The `terraform_remote_state` data source is
[documented in the Terraform Language docs](/docs/language/state/remote-state-data.html).
