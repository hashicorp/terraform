---
layout: "aws"
page_title: "AWS: aws_security_group_rule"
sidebar_current: "docs-aws-resource-security-group-rule"
description: |-
  Provides an security group rule resource.
---

# aws\_security\_group\_rule

Provides a security group rule resource. Represents a single `ingress` or
`egress` group rule, which can be added to external Security Groups.

~> **NOTE on Security Groups and Security Group Rules:** Terraform currently
provides both a standalone Security Group Rule resource (a single `ingress` or
`egress` rule), and a [Security Group resource](security_group.html) with `ingress` and `egress` rules
defined in-line. At this time you cannot use a Security Group with in-line rules
in conjunction with any Security Group Rule resources. Doing so will cause
a conflict of rule settings and will overwrite rules.

## Example Usage

Basic usage

```
resource "aws_security_group_rule" "allow_all" {
    type = "ingress"
    from_port = 0
    to_port = 65535
    protocol = "tcp"
    cidr_blocks = ["0.0.0.0/0"]

    security_group_id = "sg-123456"
    source_security_group_id = "sg-654321"
}
```

## Argument Reference

The following arguments are supported:

* `type` - (Required) The type of rule being created. Valid options are `ingress` (inbound)
or `egress` (outbound).
* `cidr_blocks` - (Optional) List of CIDR blocks.
* `from_port` - (Required) The start port (or ICMP type number if protocol is "icmp").
* `protocol` - (Required) The protocol.
* `security_group_id` - (Required) The security group to apply this rule to.
* `source_security_group_id` - (Optional) The security group id to allow access to/from,
     depending on the `type`.
* `self` - (Optional) If true, the security group itself will be added as
     a source to this ingress rule.
* `to_port` - (Required) The end range port.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the security group rule
* `type` - The type of rule, `ingress` or `egress`
* `from_port` - The source port
* `to_port` - The destination port
* `protocol` – The protocol used
