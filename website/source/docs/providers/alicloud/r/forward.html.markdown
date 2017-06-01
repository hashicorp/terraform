---
layout: "alicloud"
page_title: "Alicloud: alicloud_forward_entry"
sidebar_current: "docs-alicloud-resource-vpc"
description: |-
  Provides a Alicloud forward resource.
---

# alicloud\_forward

Provides a forward resource.

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
    "alicloud_vswitch.foo",
  ]
}

resource "alicloud_forward_entry" "foo" {
  forward_table_id = "${alicloud_nat_gateway.foo.forward_table_ids}"
  external_ip      = "${alicloud_nat_gateway.foo.bandwidth_packages.0.public_ip_addresses}"
  external_port    = "80"
  ip_protocol      = "tcp"
  internal_ip      = "172.16.0.3"
  internal_port    = "8080"
}

```
## Argument Reference

The following arguments are supported:

* `forward_table_id` - (Required, Forces new resource) The value can get from `alicloud_nat_gateway` Attributes "forward_table_ids".
* `external_ip` - (Required, Forces new resource) The external ip address, the ip must along bandwidth package public ip which `alicloud_nat_gateway` argument `bandwidth_packages`.
* `external_port` - (Required) The external port, valid value is 1~65535|any.
* `ip_protocol` - (Required) The ip protocal, valid value is tcp|udp|any.
* `internal_ip` - (Required) The internal ip, must a private ip.
* `internal_port` - (Required) The internal port, valid value is 1~65535|any.