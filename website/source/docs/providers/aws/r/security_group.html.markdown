---
layout: "aws"
page_title: "AWS: aws_security_group"
sidebar_current: "docs-aws-resource-security-group"
description: |-
  Provides a security group resource.
---

# aws\_security\_group

Provides a security group resource.

~> **NOTE on Security Groups and Security Group Rules:** Terraform currently
provides both a standalone [Security Group Rule resource](security_group_rule.html) (a single `ingress` or
`egress` rule), and a Security Group resource with `ingress` and `egress` rules
defined in-line. At this time you cannot use a Security Group with in-line rules
in conjunction with any Security Group Rule resources. Doing so will cause
a conflict of rule settings and will overwrite rules.

## Example Usage

Basic usage

```
resource "aws_security_group" "allow_all" {
  name = "allow_all"
  description = "Allow all inbound traffic"

  ingress {
      from_port = 0
      to_port = 0
      protocol = "-1"
      cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
      from_port = 0
      to_port = 0
      protocol = "-1"
      cidr_blocks = ["0.0.0.0/0"]
  }
}
```

Basic usage with tags:

```
resource "aws_security_group" "allow_all" {
  name = "allow_all"
  description = "Allow all inbound traffic"

  ingress {
      from_port = 0
      to_port = 65535
      protocol = "tcp"
      cidr_blocks = ["0.0.0.0/0"]
  }

  tags {
    Name = "allow_all"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional) The name of the security group. If omitted, Terraform will
assign a random, unique name
* `name_prefix` - (Optional) Creates a unique name beginning with the specified
  prefix. Conflicts with `name`.
* `description` - (Optional) The security group description. Defaults to "Managed by Terraform". Cannot be "".
* `ingress` - (Optional) Can be specified multiple times for each
   ingress rule. Each ingress block supports fields documented below.
* `egress` - (Optional, VPC only) Can be specified multiple times for each
      egress rule. Each egress block supports fields documented below.
* `vpc_id` - (Optional) The VPC ID.
* `tags` - (Optional) A mapping of tags to assign to the resource.

The `ingress` block supports:

* `cidr_blocks` - (Optional) List of CIDR blocks. Cannot be used with `security_groups`.
* `from_port` - (Required) The start port.
* `protocol` - (Required) The protocol. If you select a protocol of
"-1", you must specify a "from_port" and "to_port" equal to 0.
* `security_groups` - (Optional) List of security group Group Names if using
    EC2-Classic or the default VPC, or Group IDs if using a non-default VPC.
    Cannot be used with `cidr_blocks`.
* `self` - (Optional) If true, the security group itself will be added as
     a source to this ingress rule.
* `to_port` - (Required) The end range port.

The `egress` block supports:

* `cidr_blocks` - (Optional) List of CIDR blocks. Cannot be used with `security_groups`.
* `from_port` - (Required) The start port.
* `protocol` - (Required) The protocol. If you select a protocol of
"-1", you must specify a "from_port" and "to_port" equal to 0.
* `security_groups` - (Optional) List of security group Group Names if using
    EC2-Classic or the default VPC, or Group IDs if using a non-default VPC.
    Cannot be used with `cidr_blocks`.
* `self` - (Optional) If true, the security group itself will be added as
     a source to this egress rule.
* `to_port` - (Required) The end range port.

~> **NOTE on Egress rules:** By default, AWS creates an `ALLOW ALL` egress rule when creating a
new Security Group inside of a VPC. When creating a new Security
Group inside a VPC, **Terraform will remove this default rule**, and require you
specifically re-create it if you desire that rule. We feel this leads to fewer
surprises in terms of controlling your egress rules. If you desire this rule to
be in place, you can use this `egress` block:

    egress {
      from_port = 0
      to_port = 0
      protocol = "-1"
      cidr_blocks = ["0.0.0.0/0"]
    }

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the security group
* `vpc_id` - The VPC ID.
* `owner_id` - The owner ID.
* `name` - The name of the security group
* `description` - The description of the security group
* `ingress` - The ingress rules. See above for more.
* `egress` - The egress rules. See above for more.
