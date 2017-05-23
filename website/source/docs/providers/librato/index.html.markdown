---
layout: "librato"
page_title: "Provider: Librato"
sidebar_current: "docs-librato-index"
description: |-
  The Librato provider is used to interact with the resources supported by Librato. The provider needs to be configured with the proper credentials before it can be used.
---

# Librato Provider

The Librato provider is used to interact with the
resources supported by Librato. The provider needs to be configured
with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
# Configure the Librato provider
provider "librato" {
  email = "ops@company.com"
  token = "${var.librato_token}"
}

# Create a new space
resource "librato_space" "default" {
  # ...
}
```

## Argument Reference

The following arguments are supported:

* `token` - (Required) Librato API token. It must be provided, but it can also
  be sourced from the `LIBRATO_TOKEN` environment variable.
* `email` - (Required) Librato email address. It must be provided, but it can
  also be sourced from the `LIBRATO_EMAIL` environment variable.
