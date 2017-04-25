---
layout: "openstack"
page_title: "OpenStack: openstack_lb_vip_v1"
sidebar_current: "docs-openstack-resource-lb-vip-v1"
description: |-
  Manages a V1 load balancer vip resource within OpenStack.
---

# openstack\_lb\_vip_v1

Manages a V1 load balancer vip resource within OpenStack.

## Example Usage

```hcl
resource "openstack_lb_vip_v1" "vip_1" {
  name      = "tf_test_lb_vip"
  subnet_id = "12345"
  protocol  = "HTTP"
  port      = 80
  pool_id   = "67890"
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to obtain the V2 Networking client.
    A Networking client is needed to create a VIP. If omitted, the
    `OS_REGION_NAME` environment variable is used. Changing this creates a new
    VIP.

* `name` - (Required) The name of the vip. Changing this updates the name of
    the existing vip.

* `subnet_id` - (Required) The network on which to allocate the vip's address. A
    tenant can only create vips on networks authorized by policy (e.g. networks
    that belong to them or networks that are shared). Changing this creates a
    new vip.

* `protocol` - (Required)  The protocol - can be either 'TCP, 'HTTP', or
    HTTPS'. Changing this creates a new vip.

* `port` - (Required) The port on which to listen for client traffic. Changing
    this creates a new vip.

* `pool_id` - (Required) The ID of the pool with which the vip is associated.
    Changing this updates the pool_id of the existing vip.

* `tenant_id` - (Optional) The owner of the vip. Required if admin wants to
    create a vip member for another tenant. Changing this creates a new vip.

* `address` - (Optional)  The IP address of the vip. Changing this creates a new
    vip.

* `description` - (Optional) Human-readable description for the vip. Changing
    this updates the description of the existing vip.

* `persistence` - (Optional) Omit this field to prevent session persistence.
    The persistence object structure is documented below. Changing this updates
    the persistence of the existing vip.

* `conn_limit` - (Optional) The maximum number of connections allowed for the
    vip. Default is -1, meaning no limit. Changing this updates the conn_limit
    of the existing vip.

* `floating_ip` - (Optional) A *Networking* Floating IP that will be associated
    with the vip. The Floating IP must be provisioned already.

* `admin_state_up` - (Optional) The administrative state of the vip.
    Acceptable values are "true" and "false". Changing this value updates the
    state of the existing vip.

The `persistence` block supports:

* `type` - (Required) The type of persistence mode. Valid values are "SOURCE_IP",
    "HTTP_COOKIE", or "APP_COOKIE".

* `cookie_name` - (Optional) The name of the cookie if persistence mode is set
    appropriately.

## Attributes Reference

The following attributes are exported:

* `region` - See Argument Reference above.
* `name` - See Argument Reference above.
* `subnet_id` - See Argument Reference above.
* `protocol` - See Argument Reference above.
* `port` - See Argument Reference above.
* `pool_id` - See Argument Reference above.
* `tenant_id` - See Argument Reference above.
* `address` - See Argument Reference above.
* `description` - See Argument Reference above.
* `persistence` - See Argument Reference above.
* `conn_limit` - See Argument Reference above.
* `floating_ip` - See Argument Reference above.
* `admin_state_up` - See Argument Reference above.
* `port_id` - Port UUID for this VIP at associated floating IP (if any).

## Import

Load Balancer VIPs can be imported using the `id`, e.g.

```
$ terraform import openstack_lb_vip_v1.vip_1 50e16b26-89c1-475e-a492-76167182511e
```
