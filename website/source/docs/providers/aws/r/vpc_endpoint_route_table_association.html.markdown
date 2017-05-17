---
layout: "aws"
page_title: "AWS: aws_vpc_endpoint_route_table_association"
sidebar_current: "docs-aws-resource-vpc-endpoint-route-table-association"
description: |-
  Provides a resource to create an association between a VPC endpoint and routing table.
---

# aws\_vpc\_endpoint\_route\_table\_association

Provides a resource to create an association between a VPC endpoint and routing table.

~> **NOTE on VPC Endpoints and VPC Endpoint Route Table Associations:** Terraform provides
both a standalone VPC Endpoint Route Table Association (an association between a VPC endpoint
and a single `route_table_id`) and a [VPC Endpoint](vpc_endpoint.html) resource with a `route_table_ids`
attribute. Do not use the same route table ID in both a VPC Endpoint resource and a VPC Endpoint Route
Table Association resource. Doing so will cause a conflict of associations and will overwrite the association.

## Example Usage

Basic usage:

```hcl
resource "aws_vpc_endpoint_route_table_association" "private_s3" {
  vpc_endpoint_id = "${aws_vpc_endpoint.s3.id}"
  route_table_id  = "${aws_route_table.private.id}"
}
```

## Argument Reference

The following arguments are supported:

* `vpc_endpoint_id` - (Required) The ID of the VPC endpoint with which the routing table will be associated.
* `route_table_id` - (Required) The ID of the routing table to be associated with the VPC endpoint.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the association.
