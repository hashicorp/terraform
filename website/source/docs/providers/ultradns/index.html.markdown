---
layout: "ultradns"
page_title: "Provider: UltraDNS"
sidebar_current: "docs-ultradns-index"
description: |-
  The UltraDNS provider is used to interact with the resources supported by UltraDNS. The provider needs to be configured with the proper credentials before it can be used.
---

# UltraDNS Provider

The UltraDNS provider is used to interact with the
resources supported by UltraDNS. The provider needs to be configured
with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
# Configure the UltraDNS provider
provider "ultradns" {
  username = "${var.ultradns_username}"
  password = "${var.ultradns_password}"
  baseurl  = "https://test-restapi.ultradns.com/"
}

# Create a record
resource "ultradns_record" "www" {
  # ...
}
```

## Argument Reference

The following arguments are supported:

* `username` - (Required) The UltraDNS username. It must be provided, but it can also be sourced from the `ULTRADNS_USERNAME` environment variable.
* `password` - (Required) The password associated with the username. It must be provided, but it can also be sourced from the `ULTRADNS_PASSWORD` environment variable.
* `baseurl` - (Required) The base url for the UltraDNS REST API, but it can also be sourced from the `ULTRADNS_BASEURL` environment variable.
