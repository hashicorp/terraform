---
layout: "aws"
page_title: "AWS: aws_default_network_acl"
sidebar_current: "docs-aws-resource-default-network-acl"
description: |-
  Manage the default Network ACL resource.
---

# aws\_default\_network\_acl

Provides a resource to manage the default AWS Network ACL. VPC Only.

Each VPC created in AWS comes with a Default Network ACL that can be managed, but not
destroyed. **This is an advanced resource**, and has special caveats to be aware
of when using it. Please read this document in its entirety before using this
resource.

The `aws_default_network_acl` behaves differently from normal resources, in that
Terraform does not _create_ this resource, but instead attempts to "adopt" it
into management. We can do this because each VPC created has a Default Network
ACL that cannot be destroyed, and is created with a known set of default rules.

When Terraform first adopts the Default Network ACL, it **immediately removes all
rules in the ACL**. It then proceeds to create any rules specified in the
configuration. This step is required so that only the rules specified in the
configuration are created.

This resource treats its inline rules as absolute; only the rules defined
inline are created, and any additions/removals external to this resource will
result in diffs being shown. For these reasons, this resource is incompatible with the
`aws_network_acl_rule` resource.

For more information about Network ACLs, see the AWS Documentation on
[Network ACLs][aws-network-acls].

## Basic Example Usage, with default rules

The following config gives the Default Network ACL the same rules that AWS
includes, but pulls the resource under management by Terraform. This means that
any ACL rules added or changed will be detected as drift.

```hcl
resource "aws_vpc" "mainvpc" {
  cidr_block = "10.1.0.0/16"
}

resource "aws_default_network_acl" "default" {
  default_network_acl_id = "${aws_vpc.mainvpc.default_network_acl_id}"

  ingress {
    protocol   = -1
    rule_no    = 100
    action     = "allow"
    cidr_block = "0.0.0.0/0"
    from_port  = 0
    to_port    = 0
  }

  egress {
    protocol   = -1
    rule_no    = 100
    action     = "allow"
    cidr_block = "0.0.0.0/0"
    from_port  = 0
    to_port    = 0
  }
}
```

## Example config to deny all Egress traffic, allowing Ingress

The following denies all Egress traffic by omitting any `egress` rules, while
including the default `ingress` rule to allow all traffic.

```hcl
resource "aws_vpc" "mainvpc" {
  cidr_block = "10.1.0.0/16"
}

resource "aws_default_network_acl" "default" {
  default_network_acl_id = "${aws_vpc.mainvpc.default_network_acl_id}"

  ingress {
    protocol   = -1
    rule_no    = 100
    action     = "allow"
    cidr_block = "0.0.0.0/0"
    from_port  = 0
    to_port    = 0
  }
}
```

## Example config to deny all traffic to any Subnet in the Default Network ACL:

This config denies all traffic in the Default ACL. This can be useful if you
want a locked down default to force all resources in the VPC to assign a
non-default ACL.

```hcl
resource "aws_vpc" "mainvpc" {
  cidr_block = "10.1.0.0/16"
}

resource "aws_default_network_acl" "default" {
  default_network_acl_id = "${aws_vpc.mainvpc.default_network_acl_id}"

  # no rules defined, deny all traffic in this ACL
}
```

## Argument Reference

The following arguments are supported:

* `default_network_acl_id` - (Required) The Network ACL ID to manage. This
attribute is exported from `aws_vpc`, or manually found via the AWS Console.
* `subnet_ids` - (Optional) A list of Subnet IDs to apply the ACL to. See the
notes below on managing Subnets in the Default Network ACL
* `ingress` - (Optional) Specifies an ingress rule. Parameters defined below.
* `egress` - (Optional) Specifies an egress rule. Parameters defined below.
* `tags` - (Optional) A mapping of tags to assign to the resource.

Both `egress` and `ingress` support the following keys:

* `from_port` - (Required) The from port to match.
* `to_port` - (Required) The to port to match.
* `rule_no` - (Required) The rule number. Used for ordering.
* `action` - (Required) The action to take.
* `protocol` - (Required) The protocol to match. If using the -1 'all'
protocol, you must specify a from and to port of 0.
* `cidr_block` - (Optional) The CIDR block to match. This must be a
valid network mask.
* `icmp_type` - (Optional) The ICMP type to be used. Default 0.
* `icmp_code` - (Optional) The ICMP type code to be used. Default 0.

~> Note: For more information on ICMP types and codes, see here: http://www.nthelp.com/icmp.html

### Managing Subnets in the Default Network ACL

Within a VPC, all Subnets must be associated with a Network ACL. In order to
"delete" the association between a Subnet and a non-default Network ACL, the
association is destroyed by replacing it with an association between the Subnet
and the Default ACL instead.

When managing the Default Network ACL, you cannot "remove" Subnets.
Instead, they must be reassigned to another Network ACL, or the Subnet itself must be
destroyed. Because of these requirements, removing the `subnet_ids` attribute from the
configuration of a `aws_default_network_acl` resource may result in a reoccurring
plan, until the Subnets are reassigned to another Network ACL or are destroyed.

Because Subnets are by default associated with the Default Network ACL, any
non-explicit association will show up as a plan to remove the Subnet. For
example: if you have a custom `aws_network_acl` with two subnets attached, and
you remove the `aws_network_acl` resource, after successfully destroying this
resource future plans will show a diff on the managed `aws_default_network_acl`,
as those two Subnets have been orphaned by the now destroyed network acl and thus
adopted by the Default Network ACL. In order to avoid a reoccurring plan, they
will need to be reassigned, destroyed, or added to the `subnet_ids` attribute of
the `aws_default_network_acl` entry.

### Removing `aws_default_network_acl` from your configuration

Each AWS VPC comes with a Default Network ACL that cannot be deleted. The `aws_default_network_acl`
allows you to manage this Network ACL, but Terraform cannot destroy it. Removing
this resource from your configuration will remove it from your statefile and
management, **but will not destroy the Network ACL.** All Subnets associations
and ingress or egress rules will be left as they are at the time of removal. You
can resume managing them via the AWS Console.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the Default Network ACL
* `vpc_id` -  The ID of the associated VPC
* `ingress` - Set of ingress rules
* `egress` - Set of egress rules
* `subnet_ids` – IDs of associated Subnets

[aws-network-acls]: http://docs.aws.amazon.com/AmazonVPC/latest/UserGuide/VPC_ACLs.html
