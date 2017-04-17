---
layout: "dyn"
page_title: "Provider: Dyn"
sidebar_current: "docs-dyn-index"
description: |-
  The Dyn provider is used to interact with the resources supported by Dyn. The provider needs to be configured with the proper credentials before it can be used.
---

# Dyn Provider

The Dyn provider is used to interact with the
resources supported by Dyn. The provider needs to be configured
with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
# Configure the Dyn provider
provider "dyn" {
  customer_name = "${var.dyn_customer_name}"
  username      = "${var.dyn_username}"
  password      = "${var.dyn_password}"
}

# Create a record
resource "dyn_record" "www" {
  # ...
}
```

## Argument Reference

The following arguments are supported:

* `customer_name` - (Required) The Dyn customer name. It must be provided, but it can also be sourced from the `DYN_CUSTOMER_NAME` environment variable.
* `username` - (Required) The Dyn username. It must be provided, but it can also be sourced from the `DYN_USERNAME` environment variable.
* `password` - (Required) The Dyn password. It must be provided, but it can also be sourced from the `DYN_PASSWORD` environment variable.
