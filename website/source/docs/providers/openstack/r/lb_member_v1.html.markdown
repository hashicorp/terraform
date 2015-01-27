---
layout: "openstack"
page_title: "OpenStack: openstack_lb_member_v1"
sidebar_current: "docs-openstack-resource-lb-member-v1"
description: |-
  Manages a V1 load balancer member resource within OpenStack.
---

# openstack\_lb\_member_v1

Manages a V1 load balancer member resource within OpenStack.

## Example Usage

```
resource "openstack_lb_member_v1" "node_1" {
  address = "196.172.0.1"
  port = 80
  pool_id = "12345"
  admin_state_up = true
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to obtain the V2 Networking client.
    A Networking client is needed to create an LB member. If omitted, the
    `OS_REGION_NAME` environment variable is used. Changing this creates a new
    LB member.

* `address` - (Required) The IP address of the member. Changing this creates a
    new member.

* `port` - (Required) An integer representing the port on which the member is
    hosted. Changing this creates a new member.

* `pool_id` - (Required) The pool to which this member will belong. Changing
    this creates a new member.

* `admin_state_up` - (Optional) The administrative state of the member.
    Acceptable values are 'true' and 'false'. Changing this value updates the
    state of the existing member.

* `tenant_id` - (Optional) The owner of the member. Required if admin wants to
    create a pool member for another tenant. Changing this creates a new member.

## Attributes Reference

The following attributes are exported:

* `region` - See Argument Reference above.
* `address` - See Argument Reference above.
* `port` - See Argument Reference above.
* `pool_id` - See Argument Reference above.
* `admin_state_up` - See Argument Reference above.
* `tenant_id` - See Argument Reference above.
