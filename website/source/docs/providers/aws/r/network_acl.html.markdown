---
layout: "aws"
page_title: "AWS: aws_network_acl"
sidebar_current: "docs-aws-resource-network-acl"
description: |-
  Provides a network acl resource.
---

# aws\_network\_acl

Provides a network acl resource.

## Example Usage

Basic usage

```
resource "aws_network_acl" "allow_all" {
	vpc_id = "${aws_vpc.main.id}"
	egress = {
		protocol = "tcp"
		rule_no = 1
		action = "allow"
		cidr_block =  "0.0.0.0/0"
		from_port = 0
		to_port = 65535
	}

	ingress = {
		protocol = "tcp"
		rule_no = 1
		action = "allow"
		cidr_block =  "0.0.0.0/0"
		from_port = 0
		to_port = 65535
	}
	subnet_id = "${${aws_subnet.frontend.id}}"
```

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The VPC ID.
* `subnet_id` - (Optional) The AWS Subnet ID that ACL is associated with.
* `ingress` - (Optional) Opens the ingress traffic to the subnet. Can be specified multiple times for each ingress rule.Each ingress block supports fields documented below.
* `egress` - (Optional) Opens the egress traffic to the subnet. Can be specified multiple times for each egress rule.Each egress block supports fields documented below.

The `ingress` and `egress` block supports:

* `rule_no` - (Required) The rule number. ACL entries are processed in ascending order by rule number.
* `cidr_block` - (Required) The CIDR range to allow or deny.
* `from_port` - (Required) The start port.
* `to_port` - (Required) The end range port.
* `protocol` - (Required) The protocol.
* `action` - (Required) Allows or Denies traffic for given rule.


## Attributes Reference

The following attributes are exported:

* `vpc_id` - The VPC ID
* `subnet_id` - The AWS Subnet ID that ACL is associated with.
* `ingress` - The ingress rules. See above for more.
* `egress` - The egress rules. See above for more.
