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

```hcl
resource "openstack_lb_pool_v1" "pool_1" {
  name        = "tf_test_lb_pool"
  protocol    = "HTTP"
  subnet_id   = "12345"
  lb_method   = "ROUND_ROBIN"
  lb_provider = "haproxy"
  monitor_ids = ["67890"]
}
```

## Complete Load Balancing Stack Example

```
resource "openstack_networking_network_v2" "network_1" {
  name           = "network_1"
  admin_state_up = "true"
}

resource "openstack_networking_subnet_v2" "subnet_1" {
  network_id = "${openstack_networking_network_v2.network_1.id}"
  cidr       = "192.168.199.0/24"
  ip_version = 4
}

resource "openstack_compute_secgroup_v2" "secgroup_1" {
  name        = "secgroup_1"
  description = "Rules for secgroup_1"

  rule {
    from_port   = -1
    to_port     = -1
    ip_protocol = "icmp"
    cidr        = "0.0.0.0/0"
  }

  rule {
    from_port   = 80
    to_port     = 80
    ip_protocol = "tcp"
    cidr        = "0.0.0.0/0"
  }
}

resource "openstack_compute_instance_v2" "instance_1" {
  name            = "instance_1"
  security_groups = ["default", "${openstack_compute_secgroup_v2.secgroup_1.name}"]

  network {
    uuid = "${openstack_networking_network_v2.network_1.id}"
  }
}

resource "openstack_compute_instance_v2" "instance_2" {
  name            = "instance_2"
  security_groups = ["default", "${openstack_compute_secgroup_v2.secgroup_1.name}"]

  network {
    uuid = "${openstack_networking_network_v2.network_1.id}"
  }
}

resource "openstack_lb_monitor_v1" "monitor_1" {
  type           = "TCP"
  delay          = 30
  timeout        = 5
  max_retries    = 3
  admin_state_up = "true"
}

resource "openstack_lb_pool_v1" "pool_1" {
  name        = "pool_1"
  protocol    = "TCP"
  subnet_id   = "${openstack_networking_subnet_v2.subnet_1.id}"
  lb_method   = "ROUND_ROBIN"
  monitor_ids = ["${openstack_lb_monitor_v1.monitor_1.id}"]
}

resource "openstack_lb_member_v1" "member_1" {
  pool_id = "${openstack_lb_pool_v1.pool_1.id}"
  address = "${openstack_compute_instance_v2.instance_1.access_ip_v4}"
  port    = 80
}

resource "openstack_lb_member_v1" "member_2" {
  pool_id = "${openstack_lb_pool_v1.pool_1.id}"
  address = "${openstack_compute_instance_v2.instance_2.access_ip_v4}"
  port    = 80
}

resource "openstack_lb_vip_v1" "vip_1" {
  name      = "vip_1"
  subnet_id = "${openstack_networking_subnet_v2.subnet_1.id}"
  protocol  = "TCP"
  port      = 80
  pool_id   = "${openstack_lb_pool_v1.pool_1.id}"
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

* `lb_provider` - (Optional) The backend load balancing provider. For example:
    `haproxy`, `F5`, etc.

* `tenant_id` - (Optional) The owner of the pool. Required if admin wants to
    create a pool member for another tenant. Changing this creates a new pool.

* `monitor_ids` - (Optional) A list of IDs of monitors to associate with the
    pool.

* `member` - (Optional) An existing node to add to the pool. Changing this
    updates the members of the pool. The member object structure is documented
    below. Please note that the `member` block is deprecated in favor of the
    `openstack_lb_member_v1` resource.

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
* `lb_provider` - See Argument Reference above.
* `tenant_id` - See Argument Reference above.
* `monitor_id` - See Argument Reference above.
* `member` - See Argument Reference above.

## Notes

The `member` block is deprecated in favor of the `openstack_lb_member_v1` resource.

## Import

Load Balancer Pools can be imported using the `id`, e.g.

```
$ terraform import openstack_lb_pool_v1.pool_1 b255e6ba-02ad-43e6-8951-3428ca26b713
```
