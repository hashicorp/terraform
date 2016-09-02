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
Additionall, each VPC created in AWS comes with a Default Security Group that can be managed, but not
destroyed. **This is an advanced resource**, and has special caveats to be aware
of when using it. Please read this document in its entirety before using this
resource.

The `aws_default_security_group` behaves differently from normal resources, in that
Terraform does not _create_ this resource, but instead "adopts" it
into management. We can do this because these default security groups cannot be 
destroyed, and are created with a known set of default ingress/egress rules. 

When Terraform first adopts the Default Security Group, it **immediately removes all
ingress and egress rules in the ACL**. It then proceeds to create any rules specified in the 
configuration. This step is required so that only the rules specified in the 
configuration are created.

For more information about Default Security Groups, see the AWS Documentation on 
[Default Security Groups][aws-default-security-groups].

## Basic Example Usage, with default rules

The following config gives the Default Security Group the same rules that AWS 
provides by default, but pulls the resource under management by Terraform. This means that 
any ingress or egress rules added or changed will be detected as drift.

```
resource "aws_vpc" "mainvpc" {
  cidr_block = "10.1.0.0/16"
}

resource "aws_default_security_group" "default" {
  vpc_id = "${aws_vpc.mainvpc.vpc_id}"

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

```
resource "aws_vpc" "mainvpc" {
  cidr_block = "10.1.0.0/16"
}

resource "aws_default_security_group" "default" {
  vpc_id = "${aws_vpc.mainvpc.vpc_id}"

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
resources. Namely, the `name` arguement is computed, and the `name_prefix` attribute
removed. The following arguements are still supported: 

* `description` - (Optional, Forces new resource) The security group description. Defaults to
  "Managed by Terraform". Cannot be "". __NOTE__: This field maps to the AWS
  `GroupDescription` attribute, for which there is no Update API. If you'd like
  to classify your security groups in a way that can be updated, use `tags`.
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

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the security group
* `vpc_id` - The VPC ID.
* `owner_id` - The owner ID.
* `name` - The name of the security group
* `description` - The description of the security group
* `ingress` - The ingress rules. See above for more.
* `egress` - The egress rules. See above for more.

[aws-default-security-groups]: http://docs.aws.amazon.com/fr_fr/AWSEC2/latest/UserGuide/using-network-security.html#default-security-group
