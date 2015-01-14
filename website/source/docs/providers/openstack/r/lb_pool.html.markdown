---
layout: "openstack"
page_title: "OpenStack: openstack_lb_pool"
sidebar_current: "docs-openstack-resource-lb-pool"
description: |-
  Manages a load balancer pool resource within OpenStack.
---

# openstack\_lb\_pool

Manages a load balancer pool resource within OpenStack.

## Example Usage

```
resource "openstack_lb_pool" "pool_1" {
  name = "tf_test_lb_pool"
  protocol = "HTTP"
  subnet_id = "12345"
  lb_method = "ROUND_ROBIN"
  monitor_id = "67890"
}
```

## Argument Reference

The following arguments are supported:

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

## Attributes Reference

The following attributes are exported:

* `name` - See Argument Reference above.
* `protocol` - See Argument Reference above.
* `subnet_id` - See Argument Reference above.
* `lb_method` - See Argument Reference above.
* `tenant_id` - See Argument Reference above.
* `monitor_id` - See Argument Reference above.
