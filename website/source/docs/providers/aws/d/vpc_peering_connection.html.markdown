---
layout: "aws"
page_title: "AWS: aws_vpc_peering_connection"
sidebar_current: "docs-aws-datasource-vpc-peering-connection"
description: |-
    Provides details about a specific VPC peering connection.
---

# aws\_vpc\_peering\_connection

The VPC Peering Connection data source provides details about
a specific VPC peering connection.

## Example Usage

```hcl
# Declare the data source
data "aws_vpc_peering_connection" "pc" {
  vpc_id          = "${aws_vpc.foo.id}"
  peer_cidr_block = "10.0.1.0/22"
}

# Create a route table
resource "aws_route_table" "rt" {
  vpc_id = "${aws_vpc.foo.id}"
}

# Create a route
resource "aws_route" "r" {
  route_table_id            = "${aws_route_table.rt.id}"
  destination_cidr_block    = "${data.aws_vpc_peering_connection.pc.peer_cidr_block}"
  vpc_peering_connection_id = "${data.aws_vpc_peering_connection.pc.id}"
}
```

## Argument Reference

The arguments of this data source act as filters for querying the available VPC peering connection.
The given filters must match exactly one VPC peering connection whose data will be exported as attributes.

* `id` - (Optional) The ID of the specific VPC Peering Connection to retrieve.

* `status` - (Optional) The status of the specific VPC Peering Connection to retrieve.

* `vpc_id` - (Optional) The ID of the requester VPC of the specific VPC Peering Connection to retrieve.

* `owner_id` - (Optional) The AWS account ID of the owner of the requester VPC of the specific VPC Peering Connection to retrieve.

* `cidr_block` - (Optional) The CIDR block of the requester VPC of the specific VPC Peering Connection to retrieve.

* `peer_vpc_id` - (Optional) The ID of the accepter VPC of the specific VPC Peering Connection to retrieve.

* `peer_owner_id` - (Optional) The AWS account ID of the owner of the accepter VPC of the specific VPC Peering Connection to retrieve.

* `peer_cidr_block` - (Optional) The CIDR block of the accepter VPC of the specific VPC Peering Connection to retrieve.

* `filter` - (Optional) Custom filter block as described below.

* `tags` - (Optional) A mapping of tags, each pair of which must exactly match
  a pair on the desired VPC Peering Connection.

More complex filters can be expressed using one or more `filter` sub-blocks,
which take the following arguments:

* `name` - (Required) The name of the field to filter by, as defined by
  [the underlying AWS API](http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeVpcPeeringConnections.html).

* `values` - (Required) Set of values that are accepted for the given field.
  A VPC Peering Connection will be selected if any one of the given values matches.

## Attributes Reference

All of the argument attributes except `filter` are also exported as result attributes.

* `accepter` - A configuration block that describes [VPC Peering Connection]
(http://docs.aws.amazon.com/AmazonVPC/latest/PeeringGuide) options set for the accepter VPC.

* `requester` - A configuration block that describes [VPC Peering Connection]
(http://docs.aws.amazon.com/AmazonVPC/latest/PeeringGuide) options set for the requester VPC.

#### Accepter and Requester Attributes Reference

* `allow_remote_vpc_dns_resolution` - Indicates whether a local VPC can resolve public DNS hostnames to
private IP addresses when queried from instances in a peer VPC.

* `allow_classic_link_to_remote_vpc` - Indicates whether a local ClassicLink connection can communicate
with the peer VPC over the VPC peering connection.

* `allow_vpc_to_remote_classic_link` - Indicates whether a local VPC can communicate with a ClassicLink
connection in the peer VPC over the VPC peering connection.
