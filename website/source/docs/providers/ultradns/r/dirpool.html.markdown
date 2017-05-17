---
layout: "ultradns"
page_title: "UltraDNS: ultradns_dirpool"
sidebar_current: "docs-ultradns-resource-dirpool"
description: |-
  Provides an UltraDNS Directional Controller pool resource.
---

# ultradns\_dirpool

Provides an UltraDNS Directional Controller pool resource.

## Example Usage

```hcl
# Create a Directional Controller pool
resource "ultradns_dirpool" "pool" {
  zone        = "${var.ultradns_domain}"
  name        = "terraform-dirpool"
  ttl         = 300
  description = "Minimal DirPool"

  rdata {
    host = "192.168.0.10"
  }
}
```

## Argument Reference

See [related part of UltraDNS Docs](https://restapi.ultradns.com/v1/docs#post-rrset) for details about valid values.

The following arguments are supported:

* `zone` - (Required) The domain to add the record to
* `name` - (Required) The name of the record
- `type` - (Required) The Record Type of the record
* `description` - (Required) Description of the Traffic Controller pool. Valid values are strings less than 256 characters.
* `rdata` - (Required) a list of Record Data blocks, one for each member in the pool. Record Data documented below.
* `ttl` - (Optional) The TTL of the record. Default: `3600`.
* `conflict_resolve` - (Optional) String. Valid: `"GEO"` or `"IP"`. Default: `"GEO"`.
* `no_response` - (Optional) a single Record Data block, without any `host` attribute. Record Data documented below.

Record Data blocks support the following:

* `host` - (Required in `rdata`, absent in `no_response`) IPv4 address or CNAME for the pool member.
- `all_non_configured` - (Optional) Boolean. Default: `false`.
- `geo_info` - (Optional) a single Geo Info block. Geo Info documented below.
- `ip_info` - (Optional) a single IP Info block. IP Info documented below.


Geo Info blocks support the following:

- `name` - (Optional) String.
- `is_account_level` - (Optional) Boolean. Default: `false`.
- `codes` - (Optional) Set of geo code strings. Shorthand codes are expanded.

IP Info blocks support the following:

- `name` - (Optional) String.
- `is_account_level` - (Optional) Boolean. Default: `false`.
- `ips` - (Optional) Set of IP blocks. IP Info documented below.

IP blocks support the following:
- `start` - (Optional) String. IP Address. Must be paired with `end`. Conflicts with `cidr` or `address`.
- `end` - (Optional) String. IP Address. Must be paired with `start`.
- `cidr` - (Optional) String.
- `address` - (Optional) String. IP Address.

## Attributes Reference

The following attributes are exported:

* `id` - The record ID
* `hostname` - The FQDN of the record
