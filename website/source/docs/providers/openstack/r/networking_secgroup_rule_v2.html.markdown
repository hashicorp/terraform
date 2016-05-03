---
layout: "openstack"
page_title: "OpenStack: openstack_networking_secgroup_rule_v2"
sidebar_current: "docs-openstack-resource-networking-secgroup-rule-v2"
description: |-
  Manages a V2 Neutron security group rule resource within OpenStack.
---

# openstack\_networking\_secgroup\_rule_v2

Manages a V2 neutron security group rule resource within OpenStack.
Unlike Nova security groups, neutron separates the group from the rules
and also allows an admin to target a specific tenant_id.

## Example Usage

```
resource "openstack_networking_secgroup_v2" "secgroup_1" {
  name = "secgroup_1"
  description = "My neutron security group"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_1" {
  direction = "ingress"
  ethertype = "IPv4"
  protocol = "tcp"
  port_range_min = 22
  port_range_max = 22
  remote_ip_prefix = "0.0.0.0/0"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup_1.id}"
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to obtain the V2 networking client.
    A networking client is needed to create a port. If omitted, the
    `OS_REGION_NAME` environment variable is used. Changing this creates a new
    security group rule.

* `direction` - (Required) The direction of the rule, valid values are __ingress__
    or __egress__. Changing this creates a new security group rule.

* `ethertype` - (Required) The layer 3 protocol type, valid values are __IPv4__
    or __IPv6__. Changing this creates a new security group rule.

* `protocol` - (Optional) The layer 4 protocol type, valid values are __tcp__,
    __udp__ or __icmp__. This is required if you want to specify a port range.
    Changing this creates a new security group rule.

* `port_range_min` - (Optional) The lower part of the allowed port range, valid
    integer value needs to be between 1 and 65535. Changing this creates a new
    security group rule.

* `port_range_max` - (Optional) The higher part of the allowed port range, valid
    integer value needs to be between 1 and 65535. Changing this creates a new
    security group rule.

* `remote_ip_prefix` - (Optional) The remote CIDR, the value needs to be a valid
    CIDR (i.e. 192.168.0.0/16). Changing this creates a new security group rule.

* `remote_group_id` - (Optional) The remote group id, the value needs to be an
    Openstack ID of a security group in the same tenant. Changing this creates
    a new security group rule.

* `security_group_id` - (Required) The security group id the rule shoudl belong
    to, the value needs to be an Openstack ID of a security group in the same
    tenant. Changing this creates a new security group rule.

* `tenant_id` - (Optional) The owner of the security group. Required if admin
    wants to create a port for another tenant. Changing this creates a new
    security group rule.

## Attributes Reference

The following attributes are exported:

* `region` - See Argument Reference above.
* `direction` - See Argument Reference above.
* `ethertype` - See Argument Reference above.
* `protocol` - See Argument Reference above.
* `port_range_min` - See Argument Reference above.
* `port_range_max` - See Argument Reference above.
* `remote_ip_prefix` - See Argument Reference above.
* `remote_group_id` - See Argument Reference above.
* `security_group_id` - See Argument Reference above.
* `tenant_id` - See Argument Reference above.
