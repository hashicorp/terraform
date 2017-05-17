---
layout: "aws"
page_title: "AWS: aws_subnet"
sidebar_current: "docs-aws-resource-subnet"
description: |-
  Provides an VPC subnet resource.
---

# aws\_subnet

Provides an VPC subnet resource.

## Example Usage

```hcl
resource "aws_subnet" "main" {
  vpc_id     = "${aws_vpc.main.id}"
  cidr_block = "10.0.1.0/24"

  tags {
    Name = "Main"
  }
}
```

## Argument Reference

The following arguments are supported:

* `availability_zone`- (Optional) The AZ for the subnet.
* `cidr_block` - (Required) The CIDR block for the subnet.
* `ipv6_cidr_block` - (Optional) The IPv6 network range for the subnet,
    in CIDR notation. The subnet size must use a /64 prefix length.
* `map_public_ip_on_launch` -  (Optional) Specify true to indicate
    that instances launched into the subnet should be assigned
    a public IP address. Default is `false`.
* `assign_ipv6_address_on_creation` - (Optional) Specify true to indicate
    that network interfaces created in the specified subnet should be
    assigned an IPv6 address. Default is `false`
* `vpc_id` - (Required) The VPC ID.
* `tags` - (Optional) A mapping of tags to assign to the resource.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the subnet
* `availability_zone`- The AZ for the subnet.
* `cidr_block` - The CIDR block for the subnet.
* `vpc_id` - The VPC ID.
* `ipv6_association_id` - The association ID for the IPv6 CIDR block.
* `ipv6_cidr_block` - The IPv6 CIDR block.

## Import

Subnets can be imported using the `subnet id`, e.g.

```
$ terraform import aws_subnet.public_subnet subnet-9d4a7b6c
```