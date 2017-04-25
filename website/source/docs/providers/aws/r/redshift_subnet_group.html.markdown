---
layout: "aws"
page_title: "AWS: aws_redshift_subnet_group"
sidebar_current: "docs-aws-resource-redshift-subnet-group"
description: |-
  Provides a Redshift Subnet Group resource.
---

# aws\_redshift\_subnet\_group

Creates a new Amazon Redshift subnet group. You must provide a list of one or more subnets in your existing Amazon Virtual Private Cloud (Amazon VPC) when creating Amazon Redshift subnet group.

## Example Usage

```hcl
resource "aws_vpc" "foo" {
  cidr_block = "10.1.0.0/16"
}

resource "aws_subnet" "foo" {
  cidr_block        = "10.1.1.0/24"
  availability_zone = "us-west-2a"
  vpc_id            = "${aws_vpc.foo.id}"

  tags {
    Name = "tf-dbsubnet-test-1"
  }
}

resource "aws_subnet" "bar" {
  cidr_block        = "10.1.2.0/24"
  availability_zone = "us-west-2b"
  vpc_id            = "${aws_vpc.foo.id}"

  tags {
    Name = "tf-dbsubnet-test-2"
  }
}

resource "aws_redshift_subnet_group" "foo" {
  name       = "foo"
  subnet_ids = ["${aws_subnet.foo.id}", "${aws_subnet.bar.id}"]

  tags {
    environment = "Production"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Redshift Subnet group.
* `description` - (Optional) The description of the Redshift Subnet group. Defaults to "Managed by Terraform".
* `subnet_ids` - (Required) An array of VPC subnet IDs.
* `tags` - (Optional) A mapping of tags to assign to the resource.

## Attributes Reference

The following attributes are exported:

* `id` - The Redshift Subnet group ID.

## Import

Redshift subnet groups can be imported using the `name`, e.g.

```
$ terraform import aws_redshift_subnet_group.testgroup1 test-cluster-subnet-group
```
