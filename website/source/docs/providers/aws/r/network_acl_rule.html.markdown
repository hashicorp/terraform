---
layout: "aws"
page_title: "AWS: aws_network_acl_rule"
sidebar_current: "docs-aws-resource-network-acl-rule"
description: |-
  Provides an network ACL Rule resource.
---

# aws\_network\_acl\_rule

Creates an entry (a rule) in a network ACL with the specified rule number.

## Example Usage

```
resource "aws_network_acl" "bar" {
	vpc_id = "${aws_vpc.foo.id}"
}
resource "aws_network_acl_rule" "bar" {
	network_acl_id = "${aws_network_acl.bar.id}"
	rule_number = 200
	egress = false
	protocol = "tcp"
	rule_action = "allow"
	cidr_block = "0.0.0.0/0"
	from_port = 22
	to_port = 22
}
```

## Argument Reference

The following arguments are supported:

* `network_acl_id` - (Required) The ID of the network ACL.
* `rule_number` - (Required) The rule number for the entry (for example, 100). ACL entries are processed in ascending order by rule number.
* `egress` - (Optional, bool) Indicates whether this is an egress rule (rule is applied to traffic leaving the subnet). Default `false`.
* `protocol` - (Required) The protocol. A value of -1 means all protocols.
* `rule_action` - (Required) Indicates whether to allow or deny the traffic that matches the rule. Accepted values: `allow` | `deny`
* `cidr_block` - (Required) The network range to allow or deny, in CIDR notation (for example 172.16.0.0/24 ).
* `from_port` - (Optional) The from port to match.
* `to_port` - (Optional) The to port to match.
* `icmp_type` - (Optional) ICMP protocol: The ICMP type. Required if specifying ICMP for the protocol. e.g. -1
* `icmp_code` - (Optional) ICMP protocol: The ICMP code. Required if specifying ICMP for the protocol. e.g. -1

~> Note: For more information on ICMP types and codes, see here: http://www.nthelp.com/icmp.html

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the network ACL Rule

