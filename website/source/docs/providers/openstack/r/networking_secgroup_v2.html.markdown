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

```hcl
resource "openstack_networking_secgroup_v2" "secgroup_1" {
  name        = "secgroup_1"
  description = "My neutron security group"
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to obtain the V2 networking client.
    A networking client is needed to create a port. If omitted, the
    `OS_REGION_NAME` environment variable is used. Changing this creates a new
    security group.

* `name` - (Required) A unique name for the security group.

* `description` - (Optional) A unique name for the security group.

* `tenant_id` - (Optional) The owner of the security group. Required if admin
    wants to create a port for another tenant. Changing this creates a new
    security group.

* `delete_default_rules` - (Optional) Whether or not to delete the default
    egress security rules. This is `false` by default. See the below note
    for more information.

## Attributes Reference

The following attributes are exported:

* `region` - See Argument Reference above.
* `name` - See Argument Reference above.
* `description` - See Argument Reference above.
* `tenant_id` - See Argument Reference above.

## Default Security Group Rules

In most cases, OpenStack will create some egress security group rules for each
new security group. These security group rules will not be managed by
Terraform, so if you prefer to have *all* aspects of your infrastructure
managed by Terraform, set `delete_default_rules` to `true` and then create
separate security group rules such as the following:

```hcl
resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_v4" {
  direction = "egress"
  ethertype = "IPv4"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup.id}"
}

resource "openstack_networking_secgroup_rule_v2" "secgroup_rule_v6" {
  direction = "egress"
  ethertype = "IPv6"
  security_group_id = "${openstack_networking_secgroup_v2.secgroup.id}"
}
```

Please note that this behavior may differ depending on the configuration of
the OpenStack cloud. The above illustrates the current default Neutron
behavior. Some OpenStack clouds might provide additional rules and some might
not provide any rules at all (in which case the `delete_default_rules` setting
is moot).

## Import

Security Groups can be imported using the `id`, e.g.

```
$ terraform import openstack_networking_secgroup_v2.secgroup_1 38809219-5e8a-4852-9139-6f461c90e8bc
```
