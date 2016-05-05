---
layout: "openstack"
page_title: "OpenStack: openstack_networking_secgroup_v2"
sidebar_current: "docs-openstack-resource-networking-secgroup-v2"
description: |-
  Manages a V2 Neutron security group resource within OpenStack.
---

# openstack\_networking\_secgroup_v2

Manages a V2 neutron security group resource within OpenStack.
Unlike Nova security groups, neutron separates the group from the rules
and also allows an admin to target a specific tenant_id.

## Example Usage

```
resource "openstack_networking_secgroup_v2" "secgroup_1" {
  name = "secgroup_1"
  description = "My neutron security group"
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to obtain the V2 networking client.
    A networking client is needed to create a port. If omitted, the
    `OS_REGION_NAME` environment variable is used. Changing this creates a new
    security group.

* `name` - (Required) A unique name for the security group. Changing this
    creates a new security group.

* `description` - (Optional) A unique name for the security group. Changing this
    creates a new security group.

* `tenant_id` - (Optional) The owner of the security group. Required if admin
    wants to create a port for another tenant. Changing this creates a new
    security group.

## Attributes Reference

The following attributes are exported:

* `region` - See Argument Reference above.
* `name` - See Argument Reference above.
* `description` - See Argument Reference above.
* `tenant_id` - See Argument Reference above.
