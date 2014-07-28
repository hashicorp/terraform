---
layout: "aws"
page_title: "AWS: aws_route_table"
sidebar_current: "docs-aws-resource-route-table|"
---

# aws\_route\_table

Provides a resource to create a VPC routing table.

## Example Usage

```
resource "aws_route_table" "r" {
    vpc_id = "${aws_vpc.default.id}"
    route {
        cidr_block = "10.0.1.0/24"
    }
}
```

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The ID of the routing table.
* `route` - (Required) A list of route objects. Their keys are documented below.

Each route supports the following:

* `cidr_block` - (Required) The CIDR block of the route.
* `gateway_id` - (Optional) The Internet Gateway ID.
* `instance_id` - (Optional) The EC2 instance ID.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the routing table

