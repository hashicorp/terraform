---
layout: "aws"
page_title: "AWS: aws_security_group_rules"
sidebar_current: "docs-aws-resource-security-group-rules"
description: |-
  Provides a security group rules resource.
---

# aws\_security\_group\_rules

Provides a security group rules resource. Represents all the `ingress` and `egress`
rules that should exist for a given Security Group.

~> **NOTE on Security Groups and Security Group Rules:** Terraform currently provides
this standalone Security Group Rules resource
(all `ingress` and `egress` rules for a group),
standalone [Security Group Rule resources](security_group_rule.html)
(individual `ingress` or `egress` rules), and the ability to define
`ingress` and `egress` rules in-line with
a [Security Group resource](security_group.html).
At this time, you cannot combine any of these methods to define rules for the same group.
Doing so will cause a conflict of rule settings and will overwrite rules.

## Example Usage

Basic usage

```hcl
resource "aws_security_group_rules" "allow_all" {
  security_group_id = "sg-123456"

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port       = 0
    to_port         = 0
    protocol        = "-1"
    cidr_blocks     = ["0.0.0.0/0"]
    prefix_list_ids = ["pl-12c4e678"]
  }
}
```

## Argument Reference

The following arguments are supported:

* `security_group_id` - (Required) The security group to apply these rules to.
* `ingress` - (Optional) Can be specified multiple times for each
   ingress rule. Each ingress block supports fields documented below.
   If no ingress blocks are defined, then Terraform will remove all ingress rules.
* `egress` - (Optional, VPC only) Can be specified multiple times for each
   egress rule. Each egress block supports fields documented below.
   If no egress blocks are defined, then Terraform will remove all egress rules.

The `ingress` block supports:

* `cidr_blocks` - (Optional) List of CIDR blocks.
* `ipv6_cidr_blocks` - (Optional) List of IPv6 CIDR blocks.
* `from_port` - (Required) The start port (or ICMP type number if protocol is "icmp")
* `protocol` - (Required) The protocol. If you select a protocol of
"-1" (semantically equivalent to `"all"`, which is not a valid value here), you must specify a "from_port" and "to_port" equal to 0. If not icmp, tcp, udp, or "-1" use the [protocol number](https://www.iana.org/assignments/protocol-numbers/protocol-numbers.xhtml)
* `security_groups` - (Optional) List of security group Group Names if using
    EC2-Classic, or Group IDs if using a VPC.
* `self` - (Optional) If true, the security group itself will be added as
     a source to this ingress rule.
* `to_port` - (Required) The end range port (or ICMP code if protocol is "icmp").

The `egress` block supports:

* `cidr_blocks` - (Optional) List of CIDR blocks.
* `ipv6_cidr_blocks` - (Optional) List of IPv6 CIDR blocks.
* `prefix_list_ids` - (Optional) List of prefix list IDs (for allowing access to VPC endpoints)
* `from_port` - (Required) The start port (or ICMP type number if protocol is "icmp")
* `protocol` - (Required) The protocol. If you select a protocol of
"-1" (semantically equivalent to `"all"`, which is not a valid value here), you must specify a "from_port" and "to_port" equal to 0. If not icmp, tcp, udp, or "-1" use the [protocol number](https://www.iana.org/assignments/protocol-numbers/protocol-numbers.xhtml)
* `security_groups` - (Optional) List of security group Group Names if using
    EC2-Classic, or Group IDs if using a VPC.
* `self` - (Optional) If true, the security group itself will be added as
     a source to this egress rule.
* `to_port` - (Required) The end range port (or ICMP code if protocol is "icmp").

## Usage with prefix list IDs

Prefix list IDs are managed by AWS internally. Prefix list IDs
are associated with a prefix list name, or service name, that is linked to a specific region.
Prefix list IDs are exported on VPC Endpoints, so you can use this format:

```hcl
resource "aws_security_group_rules" "allow_all" {
  security_group_id = "sg-123456"
  egress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    prefix_list_ids = ["${aws_vpc_endpoint.my_endpoint.prefix_list_id}"]
  }
}

# ...
resource "aws_vpc_endpoint" "my_endpoint" {
  # ...
}
```

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the security group
* `ingress` - The ingress rules. See above for more.
* `egress` - The egress rules. See above for more.
