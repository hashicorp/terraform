---
layout: "language"
page_title: "Terraform Cloud Configuration - Terraform Settings - Configuration Language"
sidebar_current: "docs-config-terraform"
description: "The nested `cloud` block configures Terraform's integration with Terraform Cloud."
---

# Terraform Cloud Configuration

The main module of a Terraform configuration can integrate with Terraform Cloud to enable its
[CLI-driven run workflow](/docs/cloud/run/cli.html). You only need to configure these settings when you want to use Terraform CLI to interact with Terraform Cloud. Terraform Cloud ignores them when interacting with
Terraform through version control or the API.

> **Hands On:** Try the [Migrate State to Terraform Cloud](https://learn.hashicorp.com/tutorials/terraform/cloud-migrate) tutorial on HashiCorp Learn.

You can configure the Terraform Cloud CLI integration by adding a nested `cloud` block within the top-level
`terraform` block:

```hcl
terraform {
  cloud {
    organization = "example_corp"

    workspaces {
      tags = ["app"]
    }
  }
}
```

You cannot use the CLI integration and a [state backend](/docs/language/settings/backends/index.html) in the same configuration; they are mutually exclusive. A configuration can only provide one `cloud` block and the `cloud` block cannot refer to named values like input variables, locals, or data source attributes.

Refer to [Using Terraform Cloud](/docs/cli/cloud/index.html) in the Terraform CLI docs for more information.
