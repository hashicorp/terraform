---
layout: "aws"
page_title: "AWS: aws_vpn_gateway_route_propagation"
sidebar_current: "docs-aws-resource-vpn-gateway-route-propagation"
description: |-
  Requests automatic route propagation between a VPN gateway and a route table.
---

# aws_vpn_gateway_route_propagation

Requests automatic route propagation between a VPN gateway and a route table.

~> **Note:** This resource should not be used with a route table that has
the `propagating_vgws` argument set. If that argument is set, any route
propagation not explicitly listed in its value will be removed.

## Example Usage

```hcl
resource "aws_vpn_gateway_route_propagation" "example" {
  vpn_gateway_id = "${aws_vpn_gateway.example.id}"
  route_table_id = "${aws_route_table.example.id}"
}
```

## Argument Reference

The following arguments are required:

* `vpn_gateway_id` - The id of the `aws_vpn_gateway` to propagate routes from.
* `route_table_id` - The id of the `aws_route_table` to propagate routes into.

## Attributes Reference

This resource does not export any additional attributes.
