---
layout: "aws"
page_title: "AWS: aws_network_acl"
sidebar_current: "docs-aws-datasource-network-acl"
description: |-
    Provides details about a specific network ACL.
---

# aws\network\_acl

The Network ACL data source provides details about
a specific network ACL.

## Example Usage

This example shows how you might select a network ACL by name
and add a rule to it.

```
data "aws_network_acl" "foo" {
  tags {
    Name = "Foo NACL"
  }
}

resource "aws_network_acl_rule" "bar" {
  network_acl_id = "${data.aws_network_acl.foo.id}"

  rule_number = 200
  egress      = false
  protocol    = "tcp"
  rule_action = "allow"
  cidr_block  = "0.0.0.0/0"
  from_port   = 22
  to_port     = 22"
}
```

## Argument Reference

The arguments of this data source act as filters for querying the available network ACLs in the current region.
The given filters must match exactly one network ACL whose data will be exported as attributes.


* `network_acl_id` - (Optional) The ID of the desired network ACL.
* `vpc_id` - (Optional) The ID of the VPC that the desired network ACL belongs to.
* `default` - (Optional) Boolean constraint on whether the desired network ACL is
  the default network ACL for the VPC.
* `subnet_ids` - (Optional)  A list of Subnet IDs which are associated with the desired network ACL
* `filter` - (Optional) Custom filter block as described below.
* `tags` - (Optional) A mapping of tags, each pair of which must exactly match
  a pair on the desired network ACL.

More complex filters can be expressed using one or more `filter` sub-blocks,
which take the following arguments:

* `name` - (Required) The name of the field to filter by, as defined by
  [the underlying AWS API](http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeNetworkAcls.html).
* `values` - (Required) Set of values that are accepted for the given field.
  A network ACL will be selected if any one of the given values matches.

## Attributes Reference

All of the argument attributes except `filter` are also exported as result attributes.

`ingress` and `egress` rules are also exported with the following attributes:

* `from_port` - The from port of the rule.
* `to_port` - The to port of the rule.
* `rule_no` - The rule number.
* `action` - The rule's action.
* `protocol` - The rule's protocol.
* `cidr_block` - The CIDR block of the rule.
* `ipv6_cidr_block` - The IPv6 CIDR block of the rule.
* `icmp_type` - The rule's ICMP type.
* `icmp_code` - The rule's ICMP type code.
