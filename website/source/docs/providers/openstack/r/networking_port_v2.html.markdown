---
layout: "openstack"
page_title: "OpenStack: openstack_networking_port_v2"
sidebar_current: "docs-openstack-resource-networking-port-v2"
description: |-
  Manages a V2 port resource within OpenStack.
---

# openstack\_networking\_port_v2

Manages a V2 port resource within OpenStack.

## Example Usage

```
resource "openstack_networking_network_v2" "network_1" {
  name = "network_1"
  admin_state_up = "true"
}

resource "openstack_networking_port_v2" "port_1" {
  name = "port_1"
  network_id = "${openstack_networking_network_v2.network_1.id}"
  admin_state_up = "true"
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to obtain the V2 networking client.
    A networking client is needed to create a port. If omitted, the
    `OS_REGION_NAME` environment variable is used. Changing this creates a new
    port.

* `name` - (Optional) A unique name for the port. Changing this
    updates the `name` of an existing port.

* `network_id` - (Required) The ID of the network to attach the port to. Changing
    this creates a new port.

* `admin_state_up` - (Optional) Administrative up/down status for the port
    (must be "true" or "false" if provided). Changing this updates the
    `admin_state_up` of an existing port.

* `mac_address` - (Optional) Specify a specific MAC address for the port. Changing
    this creates a new port.

* `tenant_id` - (Optional) The owner of the Port. Required if admin wants
    to create a port for another tenant. Changing this creates a new port.

* `device_owner` - (Optional) The device owner of the Port. Changing this creates
    a new port.

* `security_group_ids` - (Optional) A list of security group IDs to apply to the
    port. The security groups must be specified by ID and not name (as opposed
    to how they are configured with the Compute Instance).

* `device_id` - (Optional) The ID of the device attached to the port. Changing this
    creates a new port.

* `fixed_ip` - (Optional) An array of desired IPs for this port. The structure is
    described below.


The `fixed_ip` block supports:

* `subnet_id` - (Required) Subnet in which to allocate IP address for
this port.

* `ip_address` - (Required) IP address desired in the subnet for this
port.

## Attributes Reference

The following attributes are exported:

* `region` - See Argument Reference above.
* `admin_state_up` - See Argument Reference above.
* `mac_address` - See Argument Reference above.
* `tenant_id` - See Argument Reference above.
* `device_owner` - See Argument Reference above.
* `security_groups` - See Argument Reference above.
* `device_id` - See Argument Reference above.
