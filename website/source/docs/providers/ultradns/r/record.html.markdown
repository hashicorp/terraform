---
layout: "ultradns"
page_title: "UltraDNS: ultradns_record"
sidebar_current: "docs-ultradns-resource-record"
description: |-
  Provides a UltraDNS record resource.
---

# ultradns\_record

Provides a UltraDNS record resource.

## Example Usage

```
# Add a record to the domain
resource "ultradns_record" "foobar" {
	zone = "${var.ultradns_domain}"
	name = "terraform"
	rdata = [ "192.168.0.11" ]
	type = "A"
	ttl = 3600
}
```

## Argument Reference

See [related part of UltraDNS Docs](https://restapi.ultradns.com/v1/docs#post-rrset) for details about valid values.

The following arguments are supported:

* `zone` - (Required) The domain to add the record to
* `name` - (Required) The name of the record
* `rdata` - (Required) An array containing the values of the record
* `type` - (Required) The type of the record
* `ttl` - (Optional) The TTL of the record

## Attributes Reference

The following attributes are exported:

* `id` - The record ID
* `name` - The name of the record
* `rdata` - An array containing the values of the record
* `type` - The type of the record
* `ttl` - The TTL of the record
* `zone` - The domain of the record
* `hostname` - The FQDN of the record
