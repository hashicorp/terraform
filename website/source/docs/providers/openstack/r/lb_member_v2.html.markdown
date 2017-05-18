---
layout: "openstack"
page_title: "OpenStack: openstack_lb_member_v2"
sidebar_current: "docs-openstack-resource-lb-member-v2"
description: |-
  Manages a V2 member resource within OpenStack.
---

# openstack\_lb\_member\_v2

Manages a V2 member resource within OpenStack.

## Example Usage

```hcl
resource "openstack_lb_member_v2" "member_1" {
  address       = "192.168.199.23"
  protocol_port = 8080
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to obtain the V2 Networking client.
    A Networking client is needed to create an . If omitted, the
    `OS_REGION_NAME` environment variable is used. Changing this creates a new
    member.

* `pool_id` - (Required) The id of the pool that this member will be
    assigned to.

* `subnet_id` - (Required) The subnet in which to access the member

* `name` - (Optional) Human-readable name for the member.

* `tenant_id` - (Optional) Required for admins. The UUID of the tenant who owns
    the member.  Only administrative users can specify a tenant UUID
    other than their own. Changing this creates a new member.

* `address` - (Required) The IP address of the member to receive traffic from
    the load balancer. Changing this creates a new member.

* `protocol_port` - (Required) The port on which to listen for client traffic.
    Changing this creates a new member.

* `weight` - (Optional)  A positive integer value that indicates the relative
    portion of traffic that this member should receive from the pool. For
    example, a member with a weight of 10 receives five times as much traffic
    as a member with a weight of 2.

* `admin_state_up` - (Optional) The administrative state of the member.
    A valid value is true (UP) or false (DOWN).

## Attributes Reference

The following attributes are exported:

* `id` - The unique ID for the member.
* `name` - See Argument Reference above.
* `weight` - See Argument Reference above.
* `admin_state_up` - See Argument Reference above.
* `tenant_id` - See Argument Reference above.
* `subnet_id` - See Argument Reference above.
* `pool_id` - See Argument Reference above.
* `address` - See Argument Reference above.
* `protocol_port` - See Argument Reference above.
