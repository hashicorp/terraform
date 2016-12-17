---
layout: "ns1"
page_title: "Provider: NS1"
sidebar_current: "docs-ns1-index"
description: |-
  The NS1 provider is used to interact with the resources supported by NS1. The provider needs to be configured with the proper credentials before it can be used.
---

# NS1 Provider

The NS1 provider is used to interact with the
resources supported by NS1. The provider needs to be configured
with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the NS1 provider
provider "ns1" {
    apikey = "${var.ns1_apikey}"
}

# Create a record
resource "ns1_record" "www" {
    ...
}
```

## Argument Reference

The following arguments are supported:

* `apikey` - (Required) The NS1 API key. It must be provided, but it can also be sourced from the `NS1_APIKEY` environment variable.
