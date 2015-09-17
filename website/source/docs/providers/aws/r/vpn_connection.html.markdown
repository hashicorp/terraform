---
layout: "aws"
page_title: "AWS: aws_vpn_connection"
sidebar_current: "docs-aws-resource-vpn-connection"
description: |-
  Provides a VPN connection connected to a VPC. These objects can be connected to customer gateways, and allow you to establish tunnels between your network and the VPC.
---

# aws\_vpn\_connection


Provides a VPN connection connected to a VPC. These objects can be connected to customer gateways, and allow you to establish tunnels between your network and the VPC.

## Example Usage

```
resource "aws_vpc" "vpc" {
	  cidr_block = "10.0.0.0/16"
}

resource "aws_vpn_gateway" "vpn_gateway" {
	  vpc_id = "${aws_vpc.vpc.id}"
}

resource "aws_customer_gateway" "customer_gateway" {
	  bgp_asn = 60000
	  ip_address = "172.0.0.1"
	  type = "ipsec.1"
}

resource "aws_vpn_connection" "main" {
	  vpn_gateway_id = "${aws_vpn_gateway.vpn_gateway.id}"
	  customer_gateway_id = "${aws_customer_gateway.customer_gateway.id}"
	  type = "ipsec.1"
	  static_routes_only = true
}
```

## Argument Reference

The following arguments are supported:

* `customer_gateway_id` - (Required) The ID of the customer gateway.
* `static_routes_only` - (Required) Whether the VPN connection uses static routes exclusively. Static routes must be used for devices that don't support BGP.
* `tags` - (Optional) Tags to apply to the connection.
* `type` - (Required) The type of VPN connection. The only type AWS supports at this time is "ipsec.1".
* `vpn_gateway_id` - (Required) The ID of the virtual private gateway.

## Attribute Reference

The following attributes are exported:

* `id` - The amazon-assigned ID of the VPN connection.
* `customer_gateway_configuration` - The configuration information for the VPN connection's customer gateway (in the native XML format).
* `customer_gateway_id` - The ID of the customer gateway to which the connection is attached.
* `static_routes_only` - Whether the VPN connection uses static routes exclusively.
* `tags` - Tags applied to the connection.
* `type` - The type of VPN connection.
* `vpn_gateway_id` - The ID of the virtual private gateway to which the connection is attached.
