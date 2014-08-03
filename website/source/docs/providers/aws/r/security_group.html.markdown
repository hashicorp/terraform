---
layout: "aws"
page_title: "AWS: aws_security_group"
sidebar_current: "docs-aws-resource-security-group"
---

# aws\_security\_group

Provides an security group resource.

## Example Usage

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
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the security group
* `description` - (Required) The security group description.
* `ingress` - (Required) Can be specified multiple times for each
   ingress rule. Each ingress block supports fields documented below.
* `vpc_id` - (Optional) The VPC ID.
* `owner_id` - (Optional) The AWS Owner ID.

The `ingress` block supports:

* `cidr_blocks` - (Optional) List of CIDR blocks. Cannot be used with `security_groups`.
* `from_port` - (Required) The start port.
* `protocol` - (Required) The protocol.
* `security_groups` - (Optional) List of security group IDs. Cannot be used with `cidr_blocks`.
* `to_port` - (Required) The end range port.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the security group
* `vpc_id` - The VPC ID.
* `owner_id` - The owner ID.
* `name` - The name of the security group
* `description` - The description of the security group
* `ingress` - The ingress rules. See above for more.

