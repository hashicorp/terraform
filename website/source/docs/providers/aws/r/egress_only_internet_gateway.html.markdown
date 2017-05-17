---
layout: "aws"
page_title: "AWS: aws_egress_only_internet_gateway"
sidebar_current: "docs-aws-resource-egress-only-internet-gateway"
description: |-
  Provides a resource to create a VPC Egress Only Internet Gateway.
---

# aws\_egress\_only\_internet\_gateway

[IPv6 only] Creates an egress-only Internet gateway for your VPC. 
An egress-only Internet gateway is used to enable outbound communication 
over IPv6 from instances in your VPC to the Internet, and prevents hosts 
outside of your VPC from initiating an IPv6 connection with your instance. 

## Example Usage

```hcl
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
	assign_amazon_ipv6_cidr_block = true
}

resource "aws_egress_only_internet_gateway" "foo" {
  vpc_id = "${aws_vpc.foo.id}"
}
```

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The VPC ID to create in.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the Egress Only Internet Gateway.