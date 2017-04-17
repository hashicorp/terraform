---
layout: "aws"
page_title: "AWS: aws_route_table_association"
sidebar_current: "docs-aws-resource-route-table-association"
description: |-
  Provides a resource to create an association between a subnet and routing table.
---

# aws\_route\_table\_association

Provides a resource to create an association between a subnet and routing table.

## Example Usage

```hcl
resource "aws_route_table_association" "a" {
  subnet_id      = "${aws_subnet.foo.id}"
  route_table_id = "${aws_route_table.bar.id}"
}
```

## Argument Reference

The following arguments are supported:

* `subnet_id` - (Required) The subnet ID to create an association.
* `route_table_id` - (Required) The ID of the routing table to associate with.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the association

