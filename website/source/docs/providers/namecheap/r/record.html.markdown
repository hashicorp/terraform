---
layout: "namecheap"
page_title: "Namecheap: namecheap_record"
sidebar_current: "docs-namecheap-resource-record"
description: |-
  Provides a Namecheap record resource.
---

# namecheap\_record

Provides a Namecheap record resource.

## Example Usage

```
# Add a record to the domain
resource "namecheap_record" "foobar" {
	domain = "${var.namecheap_domain}"
	hostname = "www"
	address = "example.com."
	recordType = "CNAME"
}
```

## Argument Reference

The following arguments are supported:

* `domain` - (Required) The domain to add the record to
* `hostname` - (Required) The hostname of the record
* `address` - (Required) The address of the record
* `recordType` - (Required) The type of the record
* `ttl` - (Optional) The TTL of the record. Default value is 1800
* `mxPref` - (Optional) The mxPref of the record. Default value is 10

## Attributes Reference

The following attributes are exported:

* `hostname` - The hostname of the record
* `address` - The address of the record
* `recordType` - The type of the record
* `ttl` - The TTL of the record
* `mxPref` - The priority of the record
