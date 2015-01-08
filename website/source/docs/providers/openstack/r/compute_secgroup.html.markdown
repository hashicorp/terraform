---
layout: "openstack"
page_title: "OpenStack: openstack_compute_secgroup"
sidebar_current: "docs-openstack-resource-compute-secgroup"
description: |-
  Manages a security group resource within OpenStack.
---

# openstack\_compute\_secgroup

Manages a security group resource within OpenStack.

## Example Usage

```
resource "openstack_compute_secgroup" "secgroup_1" {
  name = "my_secgroup"
  description = "my security group"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the security group. Changing this
    updates the `name` of an existing security group.

* `description` - (Required) A description for the security group. Changing this
    updates the `description` of an existing security group.

## Attributes Reference

The following attributes are exported:

* `name` - See Argument Reference above.
* `description` - See Argument Reference above.
