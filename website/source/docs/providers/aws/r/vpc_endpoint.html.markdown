---
layout: "aws"
page_title: "AWS: aws_vpc_endpoint"
sidebar_current: "docs-aws-resource-vpc-endpoint"
description: |-
  Provides a VPC Endpoint resource.
---

# aws\_vpc\_endpoint

Provides a VPC Endpoint resource.

~> **NOTE on VPC Endpoints and VPC Endpoint Route Table Associations:** Terraform provides
both a standalone [VPC Endpoint Route Table Association](vpc_endpoint_route_table_association.html)
(an association between a VPC endpoint and a single `route_table_id`) and a VPC Endpoint resource
with a `route_table_ids` attribute. Do not use the same route table ID in both a VPC Endpoint resource
and a VPC Endpoint Route Table Association resource. Doing so will cause a conflict of associations
and will overwrite the association.

## Example Usage

Basic usage:

```hcl
resource "aws_vpc_endpoint" "private-s3" {
  vpc_id       = "${aws_vpc.main.id}"
  service_name = "com.amazonaws.us-west-2.s3"
}
```

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The ID of the VPC in which the endpoint will be used.
* `service_name` - (Required) The AWS service name, in the form `com.amazonaws.region.service`.
* `policy` - (Optional) A policy to attach to the endpoint that controls access to the service.
* `route_table_ids` - (Optional) One or more route table IDs.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the VPC endpoint.
* `prefix_list_id` - The prefix list ID of the exposed service.
* `cidr_blocks` - The list of CIDR blocks for the exposed service.

## Import

VPC Endpoints can be imported using the `vpc endpoint id`, e.g.

```
$ terraform import aws_vpc_endpoint.endpoint1 vpce-3ecf2a57
```
