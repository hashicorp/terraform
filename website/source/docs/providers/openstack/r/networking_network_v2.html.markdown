---
layout: "openstack"
page_title: "OpenStack: openstack_networking_network_v2"
sidebar_current: "docs-openstack-resource-networking-network-v2"
description: |-
  Manages a V2 Neutron network resource within OpenStack.
---

# openstack\_networking\_network_v2

Manages a V2 Neutron network resource within OpenStack.

## Example Usage

```
resource "openstack_networking_network_v2" "network_1" {
  name = "tf_test_network"
  admin_state_up = "true"
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to obtain the V2 Networking client.
    A Networking client is needed to create a Neutron network. If omitted, the
    `OS_REGION_NAME` environment variable is used. Changing this creates a new
    network.

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

* `region` - See Argument Reference above.
* `name` - See Argument Reference above.
* `shared` - See Argument Reference above.
* `tenant_id` - See Argument Reference above.
* `admin_state_up` - See Argument Reference above.
