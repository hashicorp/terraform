---
layout: "openstack"
page_title: "OpenStack: openstack_compute_secgroup_v2"
sidebar_current: "docs-openstack-resource-compute-secgroup-v2"
description: |-
  Manages a V2 security group resource within OpenStack.
---

# openstack\_compute\_secgroup_v2

Manages a V2 security group resource within OpenStack.

## Example Usage

```hcl
resource "openstack_compute_secgroup_v2" "secgroup_1" {
  name        = "my_secgroup"
  description = "my security group"

  rule {
    from_port   = 22
    to_port     = 22
    ip_protocol = "tcp"
    cidr        = "0.0.0.0/0"
  }

  rule {
    from_port   = 80
    to_port     = 80
    ip_protocol = "tcp"
    cidr        = "0.0.0.0/0"
  }
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

* `rule` - (Optional) A rule describing how the security group operates. The
    rule object structure is documented below. Changing this updates the
    security group rules. As shown in the example above, multiple rule blocks
    may be used.

The `rule` block supports:

* `from_port` - (Required) An integer representing the lower bound of the port
range to open. Changing this creates a new security group rule.

* `to_port` - (Required) An integer representing the upper bound of the port
range to open. Changing this creates a new security group rule.

* `ip_protocol` - (Required) The protocol type that will be allowed. Changing
this creates a new security group rule.

* `cidr` - (Optional) Required if `from_group_id` or `self` is empty. The IP range
that will be the source of network traffic to the security group. Use 0.0.0.0/0
to allow all IP addresses. Changing this creates a new security group rule. Cannot
be combined with `from_group_id` or `self`.

* `from_group_id` - (Optional) Required if `cidr` or `self` is empty. The ID of a
group from which to forward traffic to the parent group. Changing this creates a
new security group rule. Cannot be combined with `cidr` or `self`.

* `self` - (Optional) Required if `cidr` and `from_group_id` is empty. If true,
the security group itself will be added as a source to this ingress rule. Cannot
be combined with `cidr` or `from_group_id`.

## Attributes Reference

The following attributes are exported:

* `region` - See Argument Reference above.
* `name` - See Argument Reference above.
* `description` - See Argument Reference above.
* `rule` - See Argument Reference above.

## Notes

### ICMP Rules

When using ICMP as the `ip_protocol`, the `from_port` sets the ICMP _type_ and the `to_port` sets the ICMP _code_. To allow all ICMP types, set each value to `-1`, like so:

```hcl
rule {
  from_port = -1
  to_port = -1
  ip_protocol = "icmp"
  cidr = "0.0.0.0/0"
}
```

A list of ICMP types and codes can be found [here](https://en.wikipedia.org/wiki/Internet_Control_Message_Protocol#Control_messages).

### Referencing Security Groups

When referencing a security group in a configuration (for example, a configuration creates a new security group and then needs to apply it to an instance being created in the same configuration), it is currently recommended to reference the security group by name and not by ID, like this:

```hcl
resource "openstack_compute_instance_v2" "test-server" {
  name            = "tf-test"
  image_id        = "ad091b52-742f-469e-8f3c-fd81cadf0743"
  flavor_id       = "3"
  key_pair        = "my_key_pair_name"
  security_groups = ["${openstack_compute_secgroup_v2.secgroup_1.name}"]
}
```

## Import

Security Groups can be imported using the `id`, e.g.

```
$ terraform import openstack_compute_secgroup_v2.my_secgroup 1bc30ee9-9d5b-4c30-bdd5-7f1e663f5edf
```
