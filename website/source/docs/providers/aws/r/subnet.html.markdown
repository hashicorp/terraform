---
layout: "aws"
page_title: "AWS: aws_subnet"
sidebar_current: "docs-aws-resource-subnet"
---

# aws\_subnet

Provides an VPC subnet resource.

## Example Usage

```
resource "aws_subnet" "main" {
    vpc_id = "${aws_vpc.main.id}"
    cidr_block = "10.0.1.0/16"
}
```

## Argument Reference

The following arguments are supported:

* `availability_zone`- (Optional) The AZ for the subnet.
* `cidr_block` - (Required) The CIDR block for the subnet.
* `vpc_id` - (Required) The VPC ID.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the subnet
* `availability_zone`- The AZ for the subnet.
* `cidr_block` - The CIDR block for the subnet.
* `vpc_id` - The VPC ID.

