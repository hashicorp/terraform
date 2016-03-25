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
	name = "www"
	address = "example.com."
	type = "CNAME"
}
```

## Argument Reference

The following arguments are supported:

* `domain` - (Required) The domain to add the record to
* `name` - (Required) The name of the record
* `address` - (Required) The address of the record
* `type` - (Required) The type of the record
* `ttl` - (Optional) The TTL of the record. Default value is 1800
* `mx_pref` - (Optional) The mxPref of the record. Default value is 10

## Attributes Reference

The following attributes are exported:

* `name` - The name of the record
* `address` - The address of the record
* `type` - The type of the record
* `ttl` - The TTL of the record
* `mx_pref` - The priority of the record
* `hostname` - The FQDN of the record
