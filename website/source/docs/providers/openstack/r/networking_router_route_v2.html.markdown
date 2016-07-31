---
layout: "openstack"
page_title: "OpenStack: openstack_networking_router_route_v2"
sidebar_current: "docs-openstack-resource-networking-router-route-v2"
description: |-
  Creates a routing entry on a OpenStack V2 router.
---

# openstack\_networking\_router_route_v2

Creates a routing entry on a OpenStack V2 router.

## Example Usage

```
resource "openstack_networking_router_v2" "router_1" {
  name = "router_1"
  admin_state_up = "true"
}

resource "openstack_networking_network_v2" "network_1" {
  name = "network_1"
  admin_state_up = "true"
}

resource "openstack_networking_subnet_v2" "subnet_1" {
  network_id = "${openstack_networking_network_v2.network_1.id}"
  cidr = "192.168.199.0/24"
  ip_version = 4
}

resource "openstack_networking_router_interface_v2" "int_1" {
  router_id = "${openstack_networking_router_v2.router_1.id}"
  subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
}

resource "openstack_networking_router_route_v2" "router_route_1" {
  depends_on = ["openstack_networking_router_interface_v2.int_1"]
  router_id = "${openstack_networking_router_v2.router_1.id}"
  destination_cidr = "10.0.1.0/24"
  next_hop = "192.168.199.254"
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to obtain the V2 networking client.
    A networking client is needed to configure a routing entry on a router. If omitted, the
    `OS_REGION_NAME` environment variable is used. Changing this creates a new
    routing entry.

* `router_id` - (Required) ID of the router this routing entry belongs to. Changing
    this creates a new routing entry.

* `destination_cidr` - (Required) CIDR block to match on the packet’s destination IP. Changing
    this creates a new routing entry.

* `next_hop` - (Required) IP address of the next hop gateway.  Changing
    this creates a new routing entry.

## Attributes Reference

The following attributes are exported:

* `region` - See Argument Reference above.
* `router_id` - See Argument Reference above.
* `destination_cidr` - See Argument Reference above.
* `next_hop` - See Argument Reference above.

## Notes

The `next_hop` IP address must be directly reachable from the router at the ``openstack_networking_router_route_v2``
resource creation time.  You can ensure that by explicitly specifying a dependency on the ``openstack_networking_router_interface_v2``
resource that connects the next hop to the router, as in the example above.
