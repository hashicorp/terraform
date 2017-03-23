---
layout: "tfe"
page_title: "Provider: Terraform Enterprise"
sidebar_current: "docs-tfe-index"
description: |-
  The Terraform Enterprise provider is used to interact with configuration,
  artifacts, and metadata managed by the Terraform Enterprise service.
---

# Terraform Enterprise Provider

The Terraform Enterprise provider is used to interact with resources, configuration,
artifacts, and metadata managed by [Terraform Enterprise](https://www.terraform.io/docs/providers/index.html).
The provider needs to be configured with the proper credentials before
it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the Atlas provider
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

* `address` - (Optional) Terrafrom Enterprise server endpoint. Defaults to public Terraform Enterprise.
  This is only required when using an on-premise deployment of Terraform Enterprise. This can
  also be specified with the `ATLAS_ADDRESS` shell environment variable.

* `token` - (Required) API token. This can also be specified with the
  `ATLAS_TOKEN` shell environment variable.

