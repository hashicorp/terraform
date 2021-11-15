---
layout: "language"
page_title: "Terraform Settings - Configuration Language"
sidebar_current: "docs-config-terraform"
description: "The terraform block allows you to configure Terraform behavior, including the Terraform version, backend, integration with Terraform Cloud, and required providers."
---

# Terraform Cloud

Each Terraform configuration can integrate with Terraform Cloud to enable its
[CLI-driven run workflow](/docs/cloud/run/cli.html).

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

See [Configuring Terraform Cloud](/docs/cli/configuring-terraform-cloud/index.html) for more information.
