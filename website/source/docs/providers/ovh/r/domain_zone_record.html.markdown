---
layout: "ovh"
page_title: "OVH: domain_zone_record"
sidebar_current: "docs-ovh-domain-zone-record"
description: |-
  Creates a DNS record
---

# ovh_domain_zone_record

Creates a DNS record

## Example Usage

```
resource "ovh_domain_zone_record" "test" {
    zone = "testdemo.ovh"
    subDomain = "test"
    fieldType = "A"
    ttl = "3600"
    target = "0.0.0.0"
}
```

## Argument Reference

The following arguments are supported:

* `zone` - (Required) domain you owned on OVH, like "testdemo.ovh" for example.

* `subDomain` - (Required) The name of the network.

* `fieldType` - (Required) DNS record type : A, AAAA, CNAME ...

* `ttl` - (Optional) Time to Live, default is 3600.

* `target` - (Required) target of the DNS record.

## Attributes Reference

The following attributes are exported:

* `zone` - See Argument Reference above.
* `subDomain` - See Argument Reference above.
* `filedType` - See Argument Reference above.
* `ttl` - See Argument Reference above.
* `target` - See Argument Reference above.
* `id` - reference id given by the OVH API, 
    this id will be use to delete/update the record.
