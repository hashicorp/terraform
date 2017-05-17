---
layout: "aws"
page_title: "AWS: aws_vpn_gateway_attachment"
sidebar_current: "docs-aws-resource-vpn-gateway-attachment"
description: |-
  Provides a Virtual Private Gateway attachment resource.
---

# aws\_vpn\_gateway\_attachment

Provides a Virtual Private Gateway attachment resource, allowing for an existing
hardware VPN gateway to be attached and/or detached from a VPC.

-> **Note:** The [`aws_vpn_gateway`](vpn_gateway.html)
resource can also automatically attach the Virtual Private Gateway it creates
to an existing VPC by setting the [`vpc_id`](vpn_gateway.html#vpc_id) attribute accordingly.

## Example Usage

```hcl
resource "aws_vpc" "network" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_vpn_gateway" "vpn" {
  tags {
    Name = "example-vpn-gateway"
  }
}

resource "aws_vpn_gateway_attachment" "vpn_attachment" {
  vpc_id         = "${aws_vpc.network.id}"
  vpn_gateway_id = "${aws_vpn_gateway.vpn.id}"
}
```

See [Virtual Private Cloud](http://docs.aws.amazon.com/AmazonVPC/latest/UserGuide/VPC_Introduction.html)
and [Virtual Private Gateway](http://docs.aws.amazon.com/AmazonVPC/latest/UserGuide/VPC_VPN.html) user
guides for more information.

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The ID of the VPC.
* `vpn_gateway_id` - (Required) The ID of the Virtual Private Gateway.

## Attributes Reference

The following attributes are exported:

* `vpc_id` - The ID of the VPC that Virtual Private Gateway is attached to.
* `vpn_gateway_id` - The ID of the Virtual Private Gateway.

## Import

This resource does not support importing.
