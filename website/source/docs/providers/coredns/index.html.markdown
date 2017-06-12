---
layout: "coredns"
page_title: "Provider: CoreDNS"
sidebar_current: "docs-coredns-index"
description: |-
  The CoreDNS provider is used to interact with the resources supported by CoreDNS. The provider needs to be configured with the proper credentials before it can be used.
---

# CoreDNS Provider

The CoreDNS provider is used to interact with the
resources supported by CoreDNS. The provider needs to be configured
with the proper etcd endpoint and zones before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
# Configure the CoreDNS provider
provider "coredns" {
  etcd_endpoints = "${var.coredns_etcd_enpoints}"
  zones = "${var.coredns_zones}"
}

# Create a record
resource "coredns_record" "www" {
  # ...
}
```

## Argument Reference

The following arguments are supported:

* `etcd_endpoints` - (Required) The CoreDNS etcd endpoint. It must be provided, but it can also be sourced from the `COREDNS_ETCD_ENDPOINTS` environment variable.
* `zones` - (Required) The coredns managed zones. It must be provided, but it can also be sourced from the `COREDNS_ZONES` environment variable.
