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

```
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	availability_zone = "us-west-2a"
	vpc_id = "${aws_vpc.foo.id}"
	tags {
		Name = "tf-dbsubnet-test-1"
	}
}

resource "aws_subnet" "bar" {
	cidr_block = "10.1.2.0/24"
	availability_zone = "us-west-2b"
	vpc_id = "${aws_vpc.foo.id}"
	tags {
		Name = "tf-dbsubnet-test-2"
	}
}

resource "aws_redshift_subnet_group" "foo" {
	name = "foo"
	description = "foo description"
	subnet_ids = ["${aws_subnet.foo.id}", "${aws_subnet.bar.id}"]
}
`
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Redshift Subnet group.
* `description` - (Required) The description of the Redshift Subnet group.
* `subnet_ids` - (Optional) An array of VPC subnet IDs..

## Attributes Reference

The following attributes are exported:

* `id` - The Redshift Subnet group ID.

