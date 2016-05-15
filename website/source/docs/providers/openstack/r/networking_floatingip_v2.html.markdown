---
layout: "openstack"
page_title: "OpenStack: openstack_networking_floatingip_v2"
sidebar_current: "docs-openstack-resource-networking-floatingip-v2"
description: |-
  Manages a V2 floating IP resource within OpenStack Neutron (networking).
---

# openstack\_networking\_floatingip_v2

Manages a V2 floating IP resource within OpenStack Neutron (networking)
that can be used for load balancers.
These are similar to Nova (compute) floating IP resources,
but only compute floating IPs can be used with compute instances.

## Example Usage

### Allocating a Floating IP

```
resource "openstack_networking_floatingip_v2" "floatip_1" {
  pool = "public"
}
```

### Attach a Floating IP to a Port or Instance

```
resource "openstack_networking_port_v2" "port_1" {
  name = "port_1"
  network_id = "<network uuid>"
  admin_state_up = "true"
}

resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"

  network {
    port = "${openstack_networking_port_v2.port_1.id}"
  }
}

resource "openstack_networking_floatingip_v2" "fip_1" {
  port_id = "${openstack_networking_port_v2.port_1.id}"
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to obtain the V2 Networking client.
    A Networking client is needed to create a floating IP that can be used with
    another networking resource, such as a load balancer. If omitted, the
    `OS_REGION_NAME` environment variable is used. Changing this creates a new
    floating IP (which may or may not have a different address).

* `pool` - (Required) The name of the pool from which to obtain the floating
    IP. Changing this creates a new floating IP.

* `port_id` - ID of an existing port with at least one IP address to associate with
    this floating IP.

## Attributes Reference

The following attributes are exported:

* `address` - The actual floating IP address itself.
