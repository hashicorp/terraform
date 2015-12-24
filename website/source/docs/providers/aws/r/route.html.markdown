---
layout: "aws"
page_title: "AWS: aws_route"
sidebar_current: "docs-aws-resource-route|"
description: |-
  Provides a resource to create a routing entry in a VPC routing table.
---

# aws\_route

Provides a resource to create a routing table entry (a route) in a VPC routing table.

~> **NOTE on Route Tables and Routes:** Terraform currently
provides both a standalone [Route resource](route.html) and a Route Table resource with routes
defined in-line. At this time you cannot use a Route Table with in-line routes
in conjunction with any Route resources. Doing so will cause
a conflict of rule settings and will overwrite rules.

## Example usage:

```
resource "aws_route" "r" {
    route_table_id = "rtb-4fbb3ac4"
    destination_cidr_block = "10.0.1.0/22"
    vpc_peering_connection_id = "pcx-45ff3dc1"
    depends_on = ["aws_route_table.testing"]
}
```

## Argument Reference

The following arguments are supported:

* `route_table_id` - (Required) The ID of the routing table.
* `destination_cidr_block` - (Required) The destination CIDR block.
* `vpc_peering_connection_id` - (Optional) An ID of a VPC peering connection.
* `gateway_id` - (Optional) An ID of a VPC internet gateway or a virtual private gateway.
* `nat_gateway_id` - (Optional) An ID of a VPC NAT gateway.
* `instance_id` - (Optional) An ID of a NAT instance.
* `network_interface_id` - (Optional) An ID of a network interface.

Each route must contain either a `gateway_id`, a `nat_gateway_id`, an
`instance_id` or a `vpc_peering_connection_id` or a `network_interface_id`.
Note that the default route, mapping the VPC's CIDR block to "local", is
created implicitly and cannot be specified.

## Attributes Reference

The following attributes are exported:

~> **NOTE:** Only the target type that is specified (one of the above)
will be exported as an attribute once the resource is created.

* `route_table_id` - The ID of the routing table.
* `destination_cidr_block` - The destination CIDR block.
* `vpc_peering_connection_id` - An ID of a VPC peering connection.
* `gateway_id` - An ID of a VPC internet gateway or a virtual private gateway.
* `nat_gateway_id` - An ID of a VPC NAT gateway.
* `instance_id` - An ID of a NAT instance.
* `network_interface_id` - An ID of a network interface.
