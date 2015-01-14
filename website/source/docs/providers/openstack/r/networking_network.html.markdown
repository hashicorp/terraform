---
layout: "openstack"
page_title: "OpenStack: openstack_networking_network"
sidebar_current: "docs-openstack-resource-networking-network"
description: |-
  Manages a Neutron network resource within OpenStack.
---

# openstack\_networking\_network

Manages a Neutron network resource within OpenStack.

## Example Usage

```
resource "openstack_networking_network" "network_1" {
  name = "tf_test_network"
  admin_state_up = "true"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional) The name of the network. Changing this updates the name of
    the existing network.

* `shared` - (Optional)  Specifies whether the network resource can be accessed
    by any tenant or not. Changing this updates the sharing capabalities of the
    existing network.

* `tenant_id` - (Optional) The owner of the newtork. Required if admin wants to
    create a network for another tenant. Changing this creates a new network.

* `admin_state_up` - (Optional) The administrative state of the network.
    Acceptable values are "true" and "false". Changing this value updates the
    state of the existing network.

## Attributes Reference

The following attributes are exported:

* `name` - See Argument Reference above.
* `shared` - See Argument Reference above.
* `tenant_id` - See Argument Reference above.
* `admin_state_up` - See Argument Reference above.
