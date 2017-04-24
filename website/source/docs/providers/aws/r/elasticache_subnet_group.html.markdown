---
layout: "aws"
page_title: "AWS: aws_elasticache_subnet_group"
sidebar_current: "docs-aws-resource-elasticache-subnet-group"
description: |-
  Provides an ElastiCache Subnet Group resource.
---

# aws\_elasticache\_subnet\_group

Provides an ElastiCache Subnet Group resource.

~> **NOTE:** ElastiCache Subnet Groups are only for use when working with an
ElastiCache cluster **inside** of a VPC. If you are on EC2 Classic, see the
[ElastiCache Security Group resource](elasticache_security_group.html).

## Example Usage

```hcl
resource "aws_vpc" "foo" {
  cidr_block = "10.0.0.0/16"

  tags {
    Name = "tf-test"
  }
}

resource "aws_subnet" "foo" {
  vpc_id            = "${aws_vpc.foo.id}"
  cidr_block        = "10.0.0.0/24"
  availability_zone = "us-west-2a"

  tags {
    Name = "tf-test"
  }
}

resource "aws_elasticache_subnet_group" "bar" {
  name       = "tf-test-cache-subnet"
  subnet_ids = ["${aws_subnet.foo.id}"]
}
```

## Argument Reference

The following arguments are supported:

* `name` – (Required) Name for the cache subnet group. Elasticache converts this name to lowercase.
* `description` – (Optional) Description for the cache subnet group. Defaults to "Managed by Terraform".
* `subnet_ids` – (Required) List of VPC Subnet IDs for the cache subnet group

## Attributes Reference

The following attributes are exported:

* `description`
* `name`
* `subnet_ids`


## Import

ElastiCache Subnet Groups can be imported using the `name`, e.g.

```
$ terraform import aws_elasticache_subnet_group.bar tf-test-cache-subnet
```