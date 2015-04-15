---
layout: "aws"
page_title: "AWS: aws_security_group"
sidebar_current: "docs-aws-resource-security-group"
description: |-
  Provides an security group resource.
---

# aws\_security\_group

Provides an security group resource.

## Example Usage

Basic usage

```
resource "aws_security_group" "allow_all" {
  name = "allow_all"
  description = "Allow all inbound traffic"

  ingress {
      from_port = 0
      to_port = 65535
      protocol = "-1"
      cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
      from_port = 0
      to_port = 65535
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

* `name` - (Required) The name of the security group
* `description` - (Required) The security group description.
* `ingress` - (Optional) Can be specified multiple times for each
   ingress rule. Each ingress block supports fields documented below.
* `egress` - (Optional) Can be specified multiple times for each
      egress rule. Each egress block supports fields documented below.
      VPC only.
* `vpc_id` - (Optional) The VPC ID.
* `tags` - (Optional) A mapping of tags to assign to the resource.

The `ingress` block supports:

* `cidr_blocks` - (Optional) List of CIDR blocks. Cannot be used with `security_groups`.
* `from_port` - (Required) The start port.
* `protocol` - (Required) The protocol.
* `security_groups` - (Optional) List of security group Group Names if using
    EC2-Classic or the default VPC, or Group IDs if using a non-default VPC.
    Cannot be used with `cidr_blocks`.
* `self` - (Optional) If true, the security group itself will be added as
     a source to this ingress rule.
* `to_port` - (Required) The end range port.

The `egress` block supports:

* `cidr_blocks` - (Optional) List of CIDR blocks. Cannot be used with `security_groups`.
* `from_port` - (Required) The start port.
* `protocol` - (Required) The protocol.
* `security_groups` - (Optional) List of security group Group Names if using
    EC2-Classic or the default VPC, or Group IDs if using a non-default VPC.
    Cannot be used with `cidr_blocks`.
* `self` - (Optional) If true, the security group itself will be added as
     a source to this egress rule.
* `to_port` - (Required) The end range port.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the security group
* `vpc_id` - The VPC ID.
* `owner_id` - The owner ID.
* `name` - The name of the security group
* `description` - The description of the security group
* `ingress` - The ingress rules. See above for more.
* `egress` - The egress rules. See above for more.
