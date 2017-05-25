---
layout: "terraform-enterprise"
page_title: "Provider: Terraform Enterprise"
sidebar_current: "docs-terraform-enterprise-index"
description: |-
  The Terraform Enterprise provider is used to interact with configuration,
  artifacts, and metadata managed by the Terraform Enterprise service.
---

# Terraform Enterprise Provider

The Terraform Enterprise provider is used to interact with resources,
configuration, artifacts, and metadata managed by
[Terraform Enterprise](https://www.terraform.io/docs/providers/index.html).
The provider needs to be configured with the proper credentials before it can
be used.

Use the navigation to the left to read about the available resources.

~> **Why is this called "atlas"?** Atlas was previously a commercial offering
from HashiCorp that included a full suite of enterprise products. The products
have since been broken apart into their individual products, like **Terraform
Enterprise**. While this transition is in progress, you may see references to
"atlas" in the documentation. We apologize for the inconvenience.

## Example Usage

```hcl
# Configure the Terraform Enterprise provider
provider "atlas" {
  token = "${var.atlas_token}"
}

# Fetch an artifact configuration
data "atlas_artifact" "web" {
  # ...
}
```

## Argument Reference

The following arguments are supported:

* `address` - (Optional) Terraform Enterprise server endpoint. Defaults to
  public Terraform Enterprise. This is only required when using an on-premise
  deployment of Terraform Enterprise. This can also be specified with the
  `ATLAS_ADDRESS` shell environment variable.

* `token` - (Required) API token. This can also be specified with the
  `ATLAS_TOKEN` shell environment variable.
