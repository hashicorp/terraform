---
layout: "openstack"
page_title: "OpenStack: openstack_compute_secgroup_v2"
sidebar_current: "docs-openstack-resource-compute-secgroup-2"
description: |-
  Manages a V2 security group resource within OpenStack.
---

# openstack\_compute\_secgroup_v2

Manages a V2 security group resource within OpenStack.

## Example Usage

```
resource "openstack_compute_secgroup_v2" "secgroup_1" {
  name = "my_secgroup"
  description = "my security group"
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to obtain the V2 Compute client.
    A Compute client is needed to create a security group. If omitted, the
    `OS_REGION_NAME` environment variable is used. Changing this creates a new
    security group.

* `name` - (Required) A unique name for the security group. Changing this
    updates the `name` of an existing security group.

* `description` - (Required) A description for the security group. Changing this
    updates the `description` of an existing security group.

## Attributes Reference

The following attributes are exported:

* `region` - See Argument Reference above.
* `name` - See Argument Reference above.
* `description` - See Argument Reference above.
