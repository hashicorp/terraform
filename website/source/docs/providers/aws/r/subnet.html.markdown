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

```
resource "aws_subnet" "main" {
    vpc_id = "${aws_vpc.main.id}"
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
* `map_public_ip_on_launch` -  (Optional) Specify true to indicate
    that instances launched into the subnet should be assigned
    a public IP address.
* `vpc_id` - (Required) The VPC ID.
* `tags` - (Optional) A mapping of tags to assign to the resource.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the subnet
* `availability_zone`- The AZ for the subnet.
* `cidr_block` - The CIDR block for the subnet.
* `vpc_id` - The VPC ID.

