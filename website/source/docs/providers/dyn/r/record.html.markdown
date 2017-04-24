---
layout: "dyn"
page_title: "Dyn: dyn_record"
sidebar_current: "docs-dyn-resource-record"
description: |-
  Provides a Dyn DNS record resource.
---

# dyn\_record

Provides a Dyn DNS record resource.

## Example Usage

```hcl
# Add a record to the domain
resource "dyn_record" "foobar" {
  zone  = "${var.dyn_zone}"
  name  = "terraform"
  value = "192.168.0.11"
  type  = "A"
  ttl   = 3600
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the record.
* `type` - (Required) The type of the record.
* `value` - (Required) The value of the record.
* `zone` - (Required) The DNS zone to add the record to.
* `ttl` - (Optional) The TTL of the record. Default uses the zone default.

## Attributes Reference

The following attributes are exported:

* `id` - The record ID.
* `fqdn` - The FQDN of the record, built from the `name` and the `zone`.
