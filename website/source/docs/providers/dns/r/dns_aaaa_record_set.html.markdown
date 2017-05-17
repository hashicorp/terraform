---
layout: "dns"
page_title: "DNS: dns_aaaa_record_set"
sidebar_current: "docs-dns-aaaa-record-set"
description: |-
  Creates a AAAA type DNS record set.
---

# dns_aaaa_record_set

Creates a AAAA type DNS record set.

## Example Usage

```hcl
resource "dns_aaaa_record_set" "www" {
  zone = "example.com."
  name = "www"
  addresses = [
    "fdd5:e282:43b8:5303:dead:beef:cafe:babe",
    "fdd5:e282:43b8:5303:cafe:babe:dead:beef",
  ]
  ttl = 300
}
```

## Argument Reference

The following arguments are supported:

* `zone` - (Required) DNS zone the record set belongs to. It must be an FQDN, that is, include the trailing dot.
* `name` - (Required) The name of the record set. The `zone` argument will be appended to this value to create the full record path.
* `addresses` - (Required) The IPv6 addresses this record set will point to.
* `ttl` - (Optional) The TTL of the record set. Defaults to `3600`.

## Attributes Reference

The following attributes are exported:

* `zone` - See Argument Reference above.
* `name` - See Argument Reference above.
* `addresses` - See Argument Reference above.
* `ttl` - See Argument Reference above.
