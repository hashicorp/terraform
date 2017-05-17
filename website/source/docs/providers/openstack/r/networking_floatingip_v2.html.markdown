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

```hcl
resource "openstack_networking_floatingip_v2" "floatip_1" {
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

* `port_id` - (Optional) ID of an existing port with at least one IP address to
    associate with this floating IP.

* `tenant_id` - (Optional) The target tenant ID in which to allocate the floating
    IP, if you specify this together with a port_id, make sure the target port
    belongs to the same tenant. Changing this creates a new floating IP (which
    may or may not have a different address)

* `fixed_ip` - Fixed IP of the port to associate with this floating IP. Required if
the port has multiple fixed IPs.

* `value_specs` - (Optional) Map of additional options.

## Attributes Reference

The following attributes are exported:

* `region` - See Argument Reference above.
* `pool` - See Argument Reference above.
* `address` - The actual floating IP address itself.
* `port_id` - ID of associated port.
* `tenant_id` - the ID of the tenant in which to create the floating IP.
* `fixed_ip` - The fixed IP which the floating IP maps to.

## Import

Floating IPs can be imported using the `id`, e.g.

```
$ terraform import openstack_networking_floatingip_v2.floatip_1 2c7f39f3-702b-48d1-940c-b50384177ee1
```
