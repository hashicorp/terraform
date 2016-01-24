---
layout: "openstack"
page_title: "OpenStack: openstack_lb_pool_v1"
sidebar_current: "docs-openstack-resource-lb-pool-v1"
description: |-
  Manages a V1 load balancer pool resource within OpenStack.
---

# openstack\_lb\_pool_v1

Manages a V1 load balancer pool resource within OpenStack.

## Example Usage

```
resource "openstack_lb_pool_v1" "pool_1" {
  name = "tf_test_lb_pool"
  protocol = "HTTP"
  subnet_id = "12345"
  lb_method = "ROUND_ROBIN"
  monitor_ids = ["67890"]
  member {
    address = "192.168.0.1"
    port = 80
    admin_state_up = "true"
  }
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to obtain the V2 Networking client.
    A Networking client is needed to create an LB pool. If omitted, the
    `OS_REGION_NAME` environment variable is used. Changing this creates a new
    LB pool.

* `name` - (Required) The name of the pool. Changing this updates the name of
    the existing pool.

* `protocol` - (Required)  The protocol used by the pool members, you can use
  either 'TCP, 'HTTP', or 'HTTPS'. Changing this creates a new pool.

* `subnet_id` - (Required) The network on which the members of the pool will be
    located. Only members that are on this network can be added to the pool.
    Changing this creates a new pool.

* `lb_method` - (Required) The algorithm used to distribute load between the
    members of the pool. The current specification supports 'ROUND_ROBIN' and
    'LEAST_CONNECTIONS' as valid values for this attribute.

* `tenant_id` - (Optional) The owner of the pool. Required if admin wants to
    create a pool member for another tenant. Changing this creates a new pool.

* `monitor_ids` - (Optional) A list of IDs of monitors to associate with the
    pool.

* `member` - (Optional) An existing node to add to the pool. Changing this
    updates the members of the pool. The member object structure is documented
    below.

The `member` block supports:

* `address` - (Required) The IP address of the member. Changing this creates a
new member.

* `port` - (Required) An integer representing the port on which the member is
hosted. Changing this creates a new member.

* `admin_state_up` - (Required) The administrative state of the member.
Acceptable values are 'true' and 'false'. Changing this value updates the
state of the existing member.

* `tenant_id` - (Optional) The owner of the member. Required if admin wants to
create a pool member for another tenant. Changing this creates a new member.


## Attributes Reference

The following attributes are exported:

* `region` - See Argument Reference above.
* `name` - See Argument Reference above.
* `protocol` - See Argument Reference above.
* `subnet_id` - See Argument Reference above.
* `lb_method` - See Argument Reference above.
* `tenant_id` - See Argument Reference above.
* `monitor_id` - See Argument Reference above.
* `member` - See Argument Reference above.
