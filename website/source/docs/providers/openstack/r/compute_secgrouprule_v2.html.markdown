---
layout: "openstack"
page_title: "OpenStack: openstack_compute_secgrouprule_v2"
sidebar_current: "docs-openstack-resource-compute-secgrouprule-v2"
description: |-
  Manages a V2 security group rule resource within OpenStack.
---

# openstack\_compute\_secgrouprule_v2

Manages a V2 security group rule resource within OpenStack.

## Example Usage

```
resource "openstack_compute_secgroup_v2" "secgroup_1" {
  name = "my_secgroup"
  description = "my security group"
}

resource "openstack_compute_secgrouprule_v2" "secgrouprule_1" {
    group_id = "${openstack_compute_secgroup_v2.secgroup_1.id}"
    from_port = 22
    to_port = 22
    ip_protocol = "TCP"
    cidr = "0.0.0.0/0"
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to obtain the V2 Compute client.
    A Compute client is needed to create a security group rule. If omitted, the
    `OS_REGION_NAME` environment variable is used. Changing this creates a new
    security group rule.

* `group_id` - (Required) The ID of the group to which this rule will be added.
    Changing this creates a new security group rule.

* `from_port` - (Required) An integer representing the lower bound of the port
    range to open. Changing this creates a new security group rule.

* `to_port` - (Required) An integer representing the upper bound of the port
    range to open. Changing this creates a new security group rule.

* `ip_protocol` - (Required) The protocol type that will be allowed. Changing
    this creates a new security group rule.

* `cidr` - (Optional) Required if `from_group_id` is empty. The IP range that
    will be the source of network traffic to the security group. Use 0.0.0.0./0
    to allow all IP addresses. Changing this creates a new security group rule.

* `from_group_id - (Optional) Required if `cidr` is empty. The ID of a group
    from which to forward traffic to the parent group. Changing
    this creates a new security group rule.

## Attributes Reference

The following attributes are exported:

* `region` - See Argument Reference above.
* `group_id` - See Argument Reference above.
* `from_port` - See Argument Reference above.
* `to_port` - See Argument Reference above.
* `ip_protocol` - See Argument Reference above.
* `cidr` - See Argument Reference above.
* `from_group_id` - See Argument Reference above.
