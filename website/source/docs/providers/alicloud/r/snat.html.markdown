---
layout: "alicloud"
page_title: "Alicloud: alicloud_snat_entry"
sidebar_current: "docs-alicloud-resource-vpc"
description: |-
  Provides a Alicloud snat resource.
---

# alicloud\_snat

Provides a snat resource.

## Example Usage

Basic Usage

```
resource "alicloud_vpc" "foo" {
  ...
}

resource "alicloud_vswitch" "foo" {
  ...
}

resource "alicloud_nat_gateway" "foo" {
  vpc_id = "${alicloud_vpc.foo.id}"
  spec   = "Small"
  name   = "test_foo"

  bandwidth_packages = [
    {
      ip_count  = 2
      bandwidth = 5
      zone      = ""
    },
    {
      ip_count  = 1
      bandwidth = 6
      zone      = "cn-beijing-b"
    }
  ]

  depends_on = [
    "alicloud_vswitch.foo"
  ]
}

resource "alicloud_snat_entry" "foo" {
  snat_table_id     = "${alicloud_nat_gateway.foo.snat_table_ids}"
  source_vswitch_id = "${alicloud_vswitch.foo.id}"
  snat_ip           = "${alicloud_nat_gateway.foo.bandwidth_packages.0.public_ip_addresses}"
}
```
## Argument Reference

The following arguments are supported:

* `snat_table_id` - (Required, Forces new resource) The value can get from `alicloud_nat_gateway` Attributes "snat_table_ids".
* `source_vswitch_id` - (Required, Forces new resource) The vswitch ID.
* `snat_ip` - (Required) The SNAT ip address, the ip must along bandwidth package public ip which `alicloud_nat_gateway` argument `bandwidth_packages`.
