---
layout: "aws"
page_title: "AWS: aws_vpc"
sidebar_current: "docs-aws-resource-vpc"
---

# aws\_vpc

Provides an VPC resource.

## Example Usage

```
resource "aws_vpc" "main" {
    cidr_block = "10.0.0.0/16"
}
```

## Argument Reference

The following arguments are supported:

* `cidr_block` - (Required) The CIDR block for the VPC.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the VPC
* `cidr_block` - The CIDR block of the VPC

