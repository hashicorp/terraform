---
layout: "openstack"
page_title: "OpenStack: openstack_compute_floatingip_v2"
sidebar_current: "docs-openstack-resource-compute-floatingip-v2"
description: |-
  Manages a V2 floating IP resource within OpenStack Nova (compute).
---

# openstack\_compute\_floatingip_v2

Manages a V2 floating IP resource within OpenStack Nova (compute)
that can be used for compute instances.
These are similar to Neutron (networking) floating IP resources,
but only networking floating IPs can be used with load balancers.

## Example Usage

```hcl
resource "openstack_compute_floatingip_v2" "floatip_1" {
  pool = "public"
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to obtain the V2 Compute client.
    A Compute client is needed to create a floating IP that can be used with
    a compute instance. If omitted, the `OS_REGION_NAME` environment variable
    is used. Changing this creates a new floating IP (which may or may not
    have a different address).

* `pool` - (Required) The name of the pool from which to obtain the floating
    IP. Changing this creates a new floating IP.

## Attributes Reference

The following attributes are exported:

* `region` - See Argument Reference above.
* `pool` - See Argument Reference above.
* `address` - The actual floating IP address itself.
* `fixed_ip` - The fixed IP address corresponding to the floating IP.
* `instance_id` - UUID of the compute instance associated with the floating IP.

## Import

Floating IPs can be imported using the `id`, e.g.

```
$ terraform import openstack_compute_floatingip_v2.floatip_1 89c60255-9bd6-460c-822a-e2b959ede9d2
```
