---
layout: "coredns"
page_title: "CoreDNS: coredns_record"
sidebar_current: "docs-coredns-resource-record"
description: |-
  Provides an CoreDNS record resource.
---

# coredns\_record

Provides an CoreDNS record resource.

## Example Usage

```hcl
# Add a record to the domain
resource "coredns_record" "foobar" {
  fqdn  = "foo.skydns.local"
  rdata = ["192.168.0.11"]
  type  = "A"
  ttl   = 3600
}
```

## Argument Reference

The following arguments are supported:

* `fqdn` - (Required) The fully qualified domain name
* `rdata` - (Required) An array containing the values of the record
* `type` - (Required) The type of the record
* `ttl` - (Optional) The TTL of the record

## Attributes Reference

The following attributes are exported:

* `id` - The record ID
* `fqdn` - The fqdn of the record
* `rdata` - An array containing the values of the record
* `type` - The type of the record
* `ttl` - The TTL of the record
* `hostname` - The FQDN of the record
