---
layout: "ultradns"
page_title: "UltraDNS: ultradns_rdpool"
sidebar_current: "docs-ultradns-resource-rdpool"
description: |-
  Provides an UltraDNS Resource Distribution pool resource.
---

# ultradns\_rdpool

Provides an UltraDNS Resource Distribution (RD) pool resource, which are
used to define rules for returning multiple A or AAAA records for a given owner name. Ordering can be FIXED, RANDOM or ROUND_ROBIN.

## Example Usage
```
# Create a Resource Distribution pool

resource "ultradns_rdpool" "pool" {
  zone        = "${var.ultradns_domain}"
  name        = "terraform-rdpool"
  ttl         = 600
  description = "Example RD Pool"
  order       = "ROUND_ROBIN"
  rdata       = [ "192.168.0.10", "192.168.0.11" ]
}
```

## Argument Reference

See [related part of UltraDNS Docs](https://restapi.ultradns.com/v1/docs#post-rrset) for details about valid values.

The following arguments are supported:

* `zone` - (Required) The domain to add the record to
* `name` - (Required) The name of the record
* `rdata` - (Required) list ip addresses.
* `order` - (Optional) Ordering rule, one of FIXED, RANDOM or ROUND_ROBIN. Default: 'ROUND_ROBIN'.
* `description` - (Optional) Description of the Resource Distribution pool. Valid values are strings less than 256 characters.
* `ttl` - (Optional) The TTL of the pool in seconds. Default: `3600`.

## Attributes Reference

The following attributes are exported:

* `id` - The record ID
* `hostname` - The FQDN of the record
