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
		cidr_block =  "10.3.2.3/18"
		from_port = 443
		to_port = 443
	}

	ingress {
		protocol = "tcp"
		rule_no = 1
		action = "allow"
		cidr_block =  "10.3.10.3/18"
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
* `subnet_id` - (Optional) The ID of the associated subnet.
* `ingress` - (Optional) Specifies an ingress rule. Parameters defined below.
* `egress` - (Optional) Specifies an egress rule. Parameters defined below.
* `tags` - (Optional) A mapping of tags to assign to the resource.

Both `egress` and `ingress` support the following keys:

* `from_port` - (Required) The from port to match.
* `to_port` - (Required) The to port to match.
* `rule_no` - (Required) The rule number. Used for ordering.
* `action` - (Required) The action to take.
* `protocol` - (Required) The protocol to match.
* `cidr_block` - (Optional) The CIDR block to match.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the network ACL

