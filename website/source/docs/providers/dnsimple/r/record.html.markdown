---
layout: "dnsimple"
page_title: "DNSimple: dnsimple_record"
sidebar_current: "docs-dnsimple-resource-record"
description: |-
  Provides a DNSimple record resource.
---

# dnsimple\_record

Provides a DNSimple record resource.

## Example Usage

```hcl
# Add a record to the root domain
resource "dnsimple_record" "foobar" {
  domain = "${var.dnsimple_domain}"
  name   = ""
  value  = "192.168.0.11"
  type   = "A"
  ttl    = 3600
}
```

```hcl
# Add a record to a sub-domain
resource "dnsimple_record" "foobar" {
  domain = "${var.dnsimple_domain}"
  name   = "terraform"
  value  = "192.168.0.11"
  type   = "A"
  ttl    = 3600
}
```

## Argument Reference

The following arguments are supported:

* `domain` - (Required) The domain to add the record to
* `name` - (Required) The name of the record
* `value` - (Required) The value of the record
* `type` - (Required) The type of the record
* `ttl` - (Optional) The TTL of the record
* `priority` - (Optional) The priority of the record - only useful for some record types


## Attributes Reference

The following attributes are exported:

* `id` - The record ID
* `name` - The name of the record
* `value` - The value of the record
* `type` - The type of the record
* `ttl` - The TTL of the record
* `priority` - The priority of the record
* `domain_id` - The domain ID of the record
* `hostname` - The FQDN of the record

## Import

DNSimple resources can be imported using their domain name and numeric ID, e.g.

```
$ terraform import dnsimple_record.resource_name example.com_1234
```

The numeric ID can be found in the URL when editing a record on the dnsimple web dashboard.
