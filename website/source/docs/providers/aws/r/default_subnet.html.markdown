---
layout: "aws"
page_title: "AWS: aws_default_subnet"
sidebar_current: "docs-aws-resource-default-subnet"
description: |-
  Manage a default VPC subnet resource.
---

# aws\_default\_subnet

Provides a resource to manage a [default AWS VPC subnet](http://docs.aws.amazon.com/AmazonVPC/latest/UserGuide/default-vpc.html#default-vpc-basics)
in the current region.

The `aws_default_subnet` behaves differently from normal resources, in that
Terraform does not _create_ this resource, but instead "adopts" it
into management. 

## Example Usage

Basic usage with tags:

```
resource "aws_default_subnet" "default_az1" {
  availability_zone = "us-west-2a"

	tags {
		Name = "Default subnet for us-west-2a"
	}
}
```

## Argument Reference

The arguments of an `aws_default_subnet` differ from `aws_subnet` resources.
Namely, the `availability_zone` argument is required and the `vpc_id`, `cidr_block`, `ipv6_cidr_block`,
`map_public_ip_on_launch` and `assign_ipv6_address_on_creation` arguments are computed.
The following arguments are still supported: 

* `tags` - (Optional) A mapping of tags to assign to the resource.

### Removing `aws_default_subnet` from your configuration

The `aws_default_subnet` resource allows you to manage a region's default VPC subnet,
but Terraform cannot destroy it. Removing this resource from your configuration
will remove it from your statefile and management, but will not destroy the subnet.
You can resume managing the subnet via the AWS Console.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the subnet
* `availability_zone`- The AZ for the subnet.
* `cidr_block` - The CIDR block for the subnet.
* `vpc_id` - The VPC ID.
* `ipv6_association_id` - The association ID for the IPv6 CIDR block.
* `ipv6_cidr_block` - The IPv6 CIDR block.
