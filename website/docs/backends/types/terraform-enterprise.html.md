---
layout: "backend-types"
page_title: "Backend Type: terraform enterprise"
sidebar_current: "docs-backends-types-standard-terraform-enterprise"
description: |-
  Terraform can store its state in Terraform Enterprise
---

# terraform enterprise

!> **The `atlas` backend is deprecated.** Please use the new enhanced
[remote](/docs/backends/types/remote.html) backend for storing state and running
remote operations in Terraform Cloud.

**Kind: Standard (with no locking)**

Reads and writes state from a [Terraform Enterprise](/docs/cloud/index.html)
workspace.

-> **Why is this called "atlas"?** Before it was a standalone offering,
Terraform Enterprise was part of an integrated suite of enterprise products
called Atlas. This backend predates the current version Terraform Enterprise, so
it uses the old name.

We no longer recommend using this backend, as it does not support collaboration
features like [workspace
locking](/docs/cloud/run/index.html). Please use the new enhanced
[remote](/docs/backends/types/remote.html) backend for storing state and running
remote operations in Terraform Cloud.

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
  config = {
    name = "example_corp/networking-prod"
  }
}
```

## Configuration variables

The following configuration options / environment variables are supported:

* `name` - (Required) Full name of the workspace (`<ORGANIZATION>/<WORKSPACE>`).
* `ATLAS_TOKEN`/ `access_token`  - (Optional) A Terraform Enterprise [user API
  token](/docs/cloud/users-teams-organizations/users.html#api-tokens). We
  recommend using the `ATLAS_TOKEN` environment variable rather than setting
  `access_token` in the configuration. If not set, the token will be requested
  during a `terraform init` and saved locally.
* `address` - (Optional) The URL of a Terraform Enterprise instance. Defaults to
  the SaaS version of Terraform Enterprise, at `"https://app.terraform.io"`; if
  you use a private install, provide its URL here.
