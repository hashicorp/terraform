---
layout: "aws"
page_title: "AWS: aws_default_route_table"
sidebar_current: "docs-aws-resource-default-route-table"
description: |-
  Provides a resource to manage a Default VPC Routing Table.
---

# aws\_default\_route\_table

Provides a resource to manage a Default VPC Routing Table.

Each VPC created in AWS comes with a Default Route Table that can be managed, but not
destroyed. **This is an advanced resource**, and has special caveats to be aware
of when using it. Please read this document in its entirety before using this
resource. It is recommended you **do not** use both `aws_default_route_table` to
manage the default route table **and** use the `aws_main_route_table_association`,
due to possible conflict in routes.

The `aws_default_route_table` behaves differently from normal resources, in that
Terraform does not _create_ this resource, but instead attempts to "adopt" it
into management. We can do this because each VPC created has a Default Route
Table that cannot be destroyed, and is created with a single route.

When Terraform first adopts the Default Route Table, it **immediately removes all
defined routes**. It then proceeds to create any routes specified in the
configuration. This step is required so that only the routes specified in the
configuration present in the Default Route Table.

For more information about Route Tables, see the AWS Documentation on
[Route Tables][aws-route-tables].

For more information about managing normal Route Tables in Terraform, see our
documentation on [aws_route_table][tf-route-tables].

~> **NOTE on Route Tables and Routes:** Terraform currently
provides both a standalone [Route resource](route.html) and a Route Table resource with routes
defined in-line. At this time you cannot use a Route Table with in-line routes
in conjunction with any Route resources. Doing so will cause
a conflict of rule settings and will overwrite routes.


## Example usage with tags:

```hcl
resource "aws_default_route_table" "r" {
  default_route_table_id = "${aws_vpc.foo.default_route_table_id}"

  route {
    # ...
  }

  tags {
    Name = "default table"
  }
}
```

## Argument Reference

The following arguments are supported:

* `default_route_table_id` - (Required) The ID of the Default Routing Table.
* `route` - (Optional) A list of route objects. Their keys are documented below.
* `tags` - (Optional) A mapping of tags to assign to the resource.
* `propagating_vgws` - (Optional) A list of virtual gateways for propagation.

Each route supports the following:

* `cidr_block` - (Required) The CIDR block of the route.
* `ipv6_cidr_block` - Optional) The Ipv6 CIDR block of the route
* `egress_only_gateway_id` - (Optional) The Egress Only Internet Gateway ID.
* `gateway_id` - (Optional) The Internet Gateway ID.
* `nat_gateway_id` - (Optional) The NAT Gateway ID.
* `instance_id` - (Optional) The EC2 instance ID.
* `vpc_peering_connection_id` - (Optional) The VPC Peering ID.
* `network_interface_id` - (Optional) The ID of the elastic network interface (eni) to use.

Each route must contain either a `gateway_id`, an `instance_id`, a `nat_gateway_id`, a
`vpc_peering_connection_id` or a `network_interface_id`. Note that the default route, mapping
the VPC's CIDR block to "local", is created implicitly and cannot be specified.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the routing table


[aws-route-tables]: http://docs.aws.amazon.com/AmazonVPC/latest/UserGuide/VPC_Route_Tables.html#Route_Replacing_Main_Table
[tf-route-tables]: /docs/providers/aws/r/route_table.html
