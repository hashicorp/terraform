---
layout: "aws"
page_title: "AWS: aws_subnet"
sidebar_current: "docs-aws-datasource-subnet-x"
description: |-
    Provides details about a specific VPC subnet
---

# aws\_subnet

`aws_subnet` provides details about a specific VPC subnet.

This resource can prove useful when a module accepts a subnet id as
an input variable and needs to, for example, determine the id of the
VPC that the subnet belongs to.

## Example Usage

The following example shows how one might accept a subnet id as a variable
and use this data source to obtain the data necessary to create a security
group that allows connections from hosts in that subnet.

```hcl
variable "subnet_id" {}

data "aws_subnet" "selected" {
  id = "${var.subnet_id}"
}

resource "aws_security_group" "subnet" {
  vpc_id = "${data.aws_subnet.selected.vpc_id}"

  ingress {
    cidr_blocks = ["${data.aws_subnet.selected.cidr_block}"]
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
  }
}
```

## Argument Reference

The arguments of this data source act as filters for querying the available
subnets in the current region. The given filters must match exactly one
subnet whose data will be exported as attributes.

* `availability_zone` - (Optional) The availability zone where the
  subnet must reside.

* `cidr_block` - (Optional) The cidr block of the desired subnet.

* `ipv6_cidr_block` - (Optional) The Ipv6 cidr block of the desired subnet

* `default_for_az` - (Optional) Boolean constraint for whether the desired
  subnet must be the default subnet for its associated availability zone.

* `filter` - (Optional) Custom filter block as described below.

* `id` - (Optional) The id of the specific subnet to retrieve.

* `state` - (Optional) The state that the desired subnet must have.

* `tags` - (Optional) A mapping of tags, each pair of which must exactly match
  a pair on the desired subnet.

* `vpc_id` - (Optional) The id of the VPC that the desired subnet belongs to.

More complex filters can be expressed using one or more `filter` sub-blocks,
which take the following arguments:

* `name` - (Required) The name of the field to filter by, as defined by
  [the underlying AWS API](http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeSubnets.html).

* `values` - (Required) Set of values that are accepted for the given field.
  A subnet will be selected if any one of the given values matches.

## Attributes Reference

All of the argument attributes except `filter` blocks are also exported as
result attributes. This data source will complete the data by populating
any fields that are not included in the configuration with the data for
the selected subnet.
