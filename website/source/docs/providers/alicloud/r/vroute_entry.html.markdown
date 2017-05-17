---
layout: "alicloud"
page_title: "Alicloud: alicloud_route_entry"
sidebar_current: "docs-alicloud-resource-route-entry"
description: |-
  Provides a Alicloud Route Entry resource.
---

# alicloud\_route\_entry

Provides a route entry resource.

## Example Usage

Basic Usage

```
resource "alicloud_vpc" "vpc" {
  name       = "tf_test_foo"
  cidr_block = "172.16.0.0/12"
}

resource "alicloud_route_entry" "default" {
  router_id             = "${alicloud_vpc.default.router_id}"
  route_table_id        = "${alicloud_vpc.default.router_table_id}"
  destination_cidrblock = "${var.entry_cidr}"
  nexthop_type          = "Instance"
  nexthop_id            = "${alicloud_instance.snat.id}"
}

resource "alicloud_instance" "snat" {
  // ...
}
```
## Argument Reference

The following arguments are supported:

* `router_id` - (Required, Forces new resource) The ID of the virtual router attached to Vpc.
* `route_table_id` - (Required, Forces new resource) The ID of the route table.
* `destination_cidrblock` - (Required, Forces new resource) The RouteEntry's target network segment.
* `nexthop_type` - (Required, Forces new resource) The next hop type. Available value is Instance.
* `nexthop_id` - (Required, Forces new resource) The route entry's next hop.

## Attributes Reference

The following attributes are exported:

* `router_id` - (Required, Forces new resource) The ID of the virtual router attached to Vpc.
* `route_table_id` - (Required, Forces new resource) The ID of the route table.
* `destination_cidrblock` - (Required, Forces new resource) The RouteEntry's target network segment.
* `nexthop_type` - (Required, Forces new resource) The next hop type. Available value is Instance.
* `nexthop_id` - (Required, Forces new resource) The route entry's next hop.
