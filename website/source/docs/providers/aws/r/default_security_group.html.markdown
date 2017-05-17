---
layout: "aws"
page_title: "AWS: aws_default_security_group"
sidebar_current: "docs-aws-resource-default-security-group"
description: |-
  Manage the default Security Group resource.
---

# aws\_default\_security\_group

Provides a resource to manage the default AWS Security Group.

For EC2 Classic accounts, each region comes with a Default Security Group.
Additionally, each VPC created in AWS comes with a Default Security Group that can be managed, but not
destroyed. **This is an advanced resource**, and has special caveats to be aware
of when using it. Please read this document in its entirety before using this
resource.

The `aws_default_security_group` behaves differently from normal resources, in that
Terraform does not _create_ this resource, but instead "adopts" it
into management. We can do this because these default security groups cannot be
destroyed, and are created with a known set of default ingress/egress rules.

When Terraform first adopts the Default Security Group, it **immediately removes all
ingress and egress rules in the Security Group**. It then proceeds to create any rules specified in the
configuration. This step is required so that only the rules specified in the
configuration are created.

This resource treats it's inline rules as absolute; only the rules defined
inline are created, and any additions/removals external to this resource will
result in diff shown. For these reasons, this resource is incompatible with the
`aws_security_group_rule` resource.

For more information about Default Security Groups, see the AWS Documentation on
[Default Security Groups][aws-default-security-groups].

## Basic Example Usage, with default rules

The following config gives the Default Security Group the same rules that AWS
provides by default, but pulls the resource under management by Terraform. This means that
any ingress or egress rules added or changed will be detected as drift.

```hcl
resource "aws_vpc" "mainvpc" {
  cidr_block = "10.1.0.0/16"
}

resource "aws_default_security_group" "default" {
  vpc_id = "${aws_vpc.mainvpc.id}"

  ingress {
    protocol  = -1
    self      = true
    from_port = 0
    to_port   = 0
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
```

## Example config to deny all Egress traffic, allowing Ingress

The following denies all Egress traffic by omitting any `egress` rules, while
including the default `ingress` rule to allow all traffic.

```hcl
resource "aws_vpc" "mainvpc" {
  cidr_block = "10.1.0.0/16"
}

resource "aws_default_security_group" "default" {
  vpc_id = "${aws_vpc.mainvpc.vpc}"

  ingress {
    protocol  = -1
    self      = true
    from_port = 0
    to_port   = 0
  }
}
```

## Argument Reference

The arguments of an `aws_default_security_group` differ slightly from `aws_security_group`
resources. Namely, the `name` argument is computed, and the `name_prefix` attribute
removed. The following arguments are still supported:

* `ingress` - (Optional) Can be specified multiple times for each
   ingress rule. Each ingress block supports fields documented below.
* `egress` - (Optional, VPC only) Can be specified multiple times for each
      egress rule. Each egress block supports fields documented below.
* `vpc_id` - (Optional, Forces new resource) The VPC ID. **Note that changing
the `vpc_id` will _not_ restore any default security group rules that were
modified, added, or removed.** It will be left in it's current state
* `tags` - (Optional) A mapping of tags to assign to the resource.


## Usage

With the exceptions mentioned above, `aws_default_security_group` should
identical behavior to `aws_security_group`. Please consult [AWS_SECURITY_GROUP](/docs/providers/aws/r/security_group.html)
for further usage documentation.

### Removing `aws_default_security_group` from your configuration

Each AWS VPC (or region, if using EC2 Classic) comes with a Default Security
Group that cannot be deleted. The `aws_default_security_group` allows you to
manage this Security Group, but Terraform cannot destroy it. Removing this resource
from your configuration will remove it from your statefile and management, but
will not destroy the Security Group. All ingress or egress rules will be left as
they are at the time of removal. You can resume managing them via the AWS Console.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the security group
* `vpc_id` - The VPC ID.
* `owner_id` - The owner ID.
* `name` - The name of the security group
* `description` - The description of the security group
* `ingress` - The ingress rules. See above for more.
* `egress` - The egress rules. See above for more.

[aws-default-security-groups]: http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-network-security.html#default-security-group
