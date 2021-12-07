---
layout: "language"
page_title: "Terraform Cloud Configuration - Terraform Settings - Configuration Language"
sidebar_current: "docs-config-terraform"
description: "The nested `cloud` block configures Terraform's integration with Terraform Cloud."
---

# Terraform Cloud Configuration

The main module of a Terraform configuration can integrate with Terraform Cloud to enable its
[CLI-driven run workflow](/docs/cloud/run/cli.html). These settings are only needed when
using Terraform CLI to interact with Terraform Cloud, and are ignored when interacting with
Terraform Cloud via version control or the API.

Terraform Cloud is configured with a nested `cloud` block within the top-level
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

Using the Cloud integration is mutually exclusive of declaring any [state backend](/docs/language/settings/backends/index.html); that is, a configuration
can only declare one or the other. Similar to backends...

- A configuration can only provide one cloud block.
- A cloud block cannot refer to named values (like input variables, locals, or data source attributes).

See [Using Terraform Cloud](/docs/cli/cloud/index.html)
in the Terraform CLI docs for more information.
