---
layout: "aws"
page_title: "AWS: aws_route_table"
sidebar_current: "docs-aws-datasource-route-table"
description: |-
    Provides details about a specific Route Table
---

# aws\_route\_table

`aws_route_table` provides details about a specific Route Table.

This resource can prove useful when a module accepts a Subnet id as
an input variable and needs to, for example, add a route in
the Route Table.

## Example Usage

The following example shows how one might accept a Route Table id as a variable
and use this data source to obtain the data necessary to create a route.

```hcl
variable "subnet_id" {}

data "aws_route_table" "selected" {
  subnet_id = "${var.subnet_id}"
}

resource "aws_route" "route" {
  route_table_id            = "${data.aws_route_table.selected.id}"
  destination_cidr_block    = "10.0.1.0/22"
  vpc_peering_connection_id = "pcx-45ff3dc1"
}
```

## Argument Reference

The arguments of this data source act as filters for querying the available
Route Table in the current region. The given filters must match exactly one
Route Table whose data will be exported as attributes.


* `filter` - (Optional) Custom filter block as described below.

* `route_table_id` - (Optional) The id of the specific Route Table to retrieve.

* `tags` - (Optional) A mapping of tags, each pair of which must exactly match
  a pair on the desired Route Table.

* `vpc_id` - (Optional) The id of the VPC that the desired Route Table belongs to.

* `subnet_id` - (Optional) The id of a Subnet which is connected to the Route Table (not be exported if not given in parameter).

More complex filters can be expressed using one or more `filter` sub-blocks,
which take the following arguments:

* `name` - (Required) The name of the field to filter by, as defined by
  [the underlying AWS API](http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeRouteTables.html).

* `values` - (Required) Set of values that are accepted for the given field.
  A Route Table will be selected if any one of the given values matches.

## Attributes Reference

All of the argument attributes except `filter` and `subnet_id` blocks are also exported as
result attributes. This data source will complete the data by populating
any fields that are not included in the configuration with the data for
the selected Route Table.

`routes` are also exported with the following attributes, when there are relevants:
Each route supports the following:

* `cidr_block` - The CIDR block of the route.
* `ipv6_cidr_block` - The IPv6 CIDR block of the route.
* `egress_only_gateway_id` - The ID of the Egress Only Internet Gateway.
* `gateway_id` - The Internet Gateway ID.
* `nat_gateway_id` - The NAT Gateway ID.
* `instance_id` - The EC2 instance ID.
* `vpc_peering_connection_id` - The VPC Peering ID.
* `network_interface_id` - The ID of the elastic network interface (eni) to use.


`associations` are also exported with the following attributes:

* `route_table_association_id` - The Association ID .
* `route_table_id` - The Route Table ID.
* `subnet_id` - The Subnet ID.
* `main` - If the Association due to the Main Route Table.
