---
layout: "aws"
page_title: "AWS: aws_internet_gateway"
sidebar_current: "docs-aws-resource-internet-gateway"
description: |-
  Provides a resource to create a VPC Internet Gateway.
---

# aws\_internet\_gateway

Provides a resource to create a VPC Internet Gateway.

## Example Usage

```
resource "aws_internet_gateway" "gw" {
    vpc_id = "${aws_vpc.main.id}"

    tags {
        Name = "main"
    }
}
```

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The VPC ID to create in.
* `tags` - (Optional) A mapping of tags to assign to the resource.

-> **Note:** It's recommended to denote that the AWS Instance or Elastic IP depends on the Internet Gateway. For example:  


    resource "aws_internet_gateway" "gw" {
      vpc_id = "${aws_vpc.main.id}"
    }

    resource "aws_instance" "foo" {  
      depends_on = ["aws_internet_gateway.gw"]  
    }


## Attributes Reference

The following attributes are exported:

* `id` - The ID of the Internet Gateway.

