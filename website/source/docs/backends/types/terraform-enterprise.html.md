---
layout: "backend-types"
page_title: "Backend Type: terraform enterprise"
sidebar_current: "docs-backends-types-standard-terraform-enterprise"
description: |-
  Terraform can store the state in Terraform Enterprise
---

# terraform enterprise

**Kind: Standard (with no locking)**

Stores the state in [Terraform Enterprise](https://www.terraform.io/docs/providers/index.html).

You can create a new environment in the
Environments section and generate new token in the Tokens page under Settings.

~> **Why is this called "atlas"?** Atlas was previously a commercial offering
from HashiCorp that included a full suite of enterprise products. The products
have since been broken apart into their individual products, like **Terraform
Enterprise**. While this transition is in progress, you may see references to
"atlas" in the documentation. We apologize for the inconvenience.

## Example Configuration

```hcl
terraform {
  backend "atlas" {
    name         = "bigbang/example"
    access_token = "foo"
  }
}
```

Note that for the access token we recommend using a
[partial configuration](/docs/backends/config.html).

## Example Referencing

```hcl
data "terraform_remote_state" "foo" {
  backend = "atlas"
  config {
    name         = "bigbang/example"
    access_token = "X2iTFefU5aWOjg.atlasv1.YaDa"
  }
}
```

## Configuration variables

The following configuration options / environment variables are supported:

 * `name` - (Required) Full name of the environment (`<username>/<name>`)
 * `access_token` / `ATLAS_TOKEN` - (Required) Terraform Enterprise API token
 * `address` - (Optional) Address to alternative Terraform Enterprise location (Terraform Enterprise endpoint)
