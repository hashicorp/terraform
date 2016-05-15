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

### Allocating a Floating IP

```
resource "openstack_compute_floatingip_v2" "floatip_1" {
  pool = "public"
}
```

### Attaching a Floating IP to an Instance

```
resource "openstack_compute_instance_v2" "instance_1" {
  name = "instance_1"
  security_groups = ["default"]

  network {
    name = "my_network"
  }
}

resource "openstack_compute_floatingip_v2" "floatip_1" {
  pool = "public"
  instance_id = "${openstack_compute_instance_v2.instance_1.id}"
  fixed_ip = "${openstack_compute_instance_v2.instance_1.access_ip_v4}"
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

* `instance_id` - (Optional; Required with `fixed_ip`) The ID of the instance
    to attach the floating IP.

* `fixed_ip` - (Optional; Required with `instance_id`) The Fixed IP of the
    instance to attach the floating IP.

## Attributes Reference

The following attributes are exported:

* `region` - See Argument Reference above.
* `pool` - See Argument Reference above.
* `address` - The actual floating IP address itself.
