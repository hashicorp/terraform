---
layout: "aws"
page_title: "AWS: aws_network_acl"
sidebar_current: "docs-aws-resource-network-acl"
description: |-
  Provides an network ACL resource.
---

# aws\_network\_acl

Provides an network ACL resource. You might set up network ACLs with rules similar
to your security groups in order to add an additional layer of security to your VPC.

## Example Usage

```
resource "aws_network_acl" "main" {
	vpc_id = "${aws_vpc.main.id}"
	egress {
		protocol = "tcp"
		rule_no = 2
		action = "allow"
		cidr_block =  "10.3.0.0/18"
		from_port = 443
		to_port = 443
	}

	ingress {
		protocol = "tcp"
		rule_no = 1
		action = "allow"
		cidr_block =  "10.3.0.0/18"
		from_port = 80
		to_port = 80
	}

	tags {
		Name = "main"
	}
}
```

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The ID of the associated VPC.
* `subnet_ids` - (Optional) A list of Subnet IDs to apply the ACL to
* `subnet_id` - (Optional, Deprecated) The ID of the associated Subnet. This
attribute is deprecated, please use the `subnet_ids` attribute instead
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

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the network ACL

