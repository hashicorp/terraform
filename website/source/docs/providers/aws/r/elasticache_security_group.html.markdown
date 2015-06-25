---
layout: "aws"
page_title: "AWS: aws_elasticache_security_group"
sidebar_current: "docs-aws-resource-elasticache-security-group"
description: |-
  Provides an ElastiCache Security Group to control access to one or more cache clusters.
---

# aws\_elasticache\_security\_<wbr>group

Provides an ElastiCache Security Group to control access to one or more cache 
clusters.

~> **NOTE:** ElastiCache Security Groups are for use only when working with an
ElastiCache cluster **outside** of a VPC. If you are using a VPC, see the
[ElastiCache Subnet Group resource](elasticache_subnet_group.html).

## Example Usage

```
resource "aws_security_group" "bar" {
    name = "security-group"
    description = "security group"
}

resource "aws_elasticache_security_group" "bar" {
    name = "elasticache-security-group"
    description = "elasticache security group"
    security_group_names = ["${aws_security_group.bar.name}"]
}
```

## Argument Reference

The following arguments are supported:

* `description` – (Required) description for the cache security group
* `name` – (Required) Name for the cache security group. This value is stored as 
a lowercase string
* `security_group_names` – (Required) List of EC2 security group names to be 
authorized for ingress to the cache security group


## Attributes Reference

The following attributes are exported:

* `description`
* `name`
* `security_group_names`
