---
layout: "ns1"
page_title: "NS1: ns1_zone"
sidebar_current: "docs-ns1-resource-zone"
description: |-
  Provides a NS1 Zone resource.
---

# ns1\_zone

Provides a NS1 DNS Zone resource. This can be used to create, modify, and delete zones.

## Example Usage

```hcl
# Create a new DNS zone
resource "ns1_zone" "example" {
  zone = "terraform.example.io"
  ttl  = 600
}
```

## Argument Reference

The following arguments are supported:

* `zone` - (Required) The domain name of the zone.
* `link` - (Optional) The target zone(domain name) to link to.
* `ttl` - (Optional) The SOA TTL.
* `refresh` - (Optional) The SOA Refresh.
* `retry` - (Optional) The SOA Retry.
* `expiry` - (Optional) The SOA Expiry.
* `nx_ttl` - (Optional) The SOA NX TTL.
* `primary` - (Optional) The primary zones' ip. This makes the zone a secondary.
