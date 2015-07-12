---
layout: "aws"
page_title: "AWS: aws_vpn_gateway"
sidebar_current: "docs-aws-resource-vpn-gateway"
description: |-
  Provides a resource to create a VPC VPN Gateway.
---

# aws\_vpn\_gateway

Provides a resource to create a VPC VPN Gateway.

## Example Usage

```
resource "aws_vpn_gateway" "vpn_gw" {
    vpc_id = "${aws_vpc.main.id}"

    tags {
        Name = "main"
    }
}
```

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Optional) The VPC ID to create in.
* `availability_zone` - (Optional) The Availability Zone for the virtual private gateway.
* `tags` - (Optional) A mapping of tags to assign to the resource.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the VPN Gateway.

