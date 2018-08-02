---
layout: "backend-types"
page_title: "Backend Type: terraform enterprise"
sidebar_current: "docs-backends-types-standard-terraform-enterprise"
description: |-
  Terraform can store the state in Terraform Enterprise
---

# terraform enterprise

**Kind: Standard (with no locking)**

Reads and writes state from a [Terraform Enterprise](/docs/enterprise/index.html)
workspace.

-> **Why is this called "atlas"?** Before it was a standalone offering,
Terraform Enterprise was part of an integrated suite of enterprise products
called Atlas. This backend predates the current version Terraform Enterprise, so
it uses the old name.

This backend is useful for uncommon tasks like migrating state into a Terraform
Enterprise workspace, but we no longer recommend using it as part of your
day-to-day Terraform workflow. Since it performs runs outside of Terraform
Enterprise and updates state directly, it does not support Terraform
Enterprise's collaborative features like [workspace
locking](/docs/enterprise/run/index.html). To perform Terraform Enterprise runs
from the command line, use [Terraform Enterprise's CLI-driven
workflow](/docs/enterprise/run/cli.html) instead.

## Example Configuration

```hcl
terraform {
  backend "atlas" {
    name = "example_corp/networking-prod"
    address = "https://app.terraform.io" # optional
  }
}
```

We recommend using a [partial configuration](/docs/backends/config.html) and
omitting the access token, which can be provided as an environment variable.

## Example Referencing

```hcl
data "terraform_remote_state" "foo" {
  backend = "atlas"
  config {
    name = "example_corp/networking-prod"
  }
}
```

## Configuration variables

The following configuration options / environment variables are supported:

* `name` - (Required) Full name of the workspace (`<ORGANIZATION>/<WORKSPACE>`).
* `ATLAS_TOKEN`/ `access_token`  - (Required) A Terraform Enterprise [user API
  token](/docs/enterprise/users-teams-organizations/users.html#api-tokens). We
  recommend using the `ATLAS_TOKEN` environment variable rather than setting
  `access_token` in the configuration.
* `address` - (Optional) The URL of a Terraform Enterprise instance. Defaults to
  the SaaS version of Terraform Enterprise, at `"https://app.terraform.io"`; if
  you use a private install, provide its URL here.
