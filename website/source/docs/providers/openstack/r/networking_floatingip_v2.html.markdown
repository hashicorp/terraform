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

```
resource "openstack_networking_floatingip_v2" "floatip_1" {
  region = ""
  pool = "public"
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

* `region` - See Argument Reference above.
* `pool` - See Argument Reference above.
* `address` - The actual floating IP address itself.
* `port_id` - ID of associated port.
