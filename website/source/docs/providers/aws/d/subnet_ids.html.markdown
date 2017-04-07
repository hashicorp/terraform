---
layout: "aws"
page_title: "AWS: aws_subnet_ids"
sidebar_current: "docs-aws-datasource-subnet-ids"
description: |-
    Provides a list of subnet Ids for a VPC
---

# aws\_subnet\_ids

`aws_subnet_ids` provides a list of ids for a vpc_id

This resource can be useful for getting back a list of subnet ids for a vpc.

## Example Usage

The following shows outputing all cidr blocks for every subnet id in a vpc.

```hcl
data "aws_subnet_ids" "example" {
  vpc_id = "${var.vpc_id}"
}

data "aws_subnet" "example" {
  count = "${length(data.aws_subnet_ids.example.ids)}"
  id = "${aws_subnet_ids.example.ids[count.index]}"
}

output "subnet_cidr_blocks" {
  value = ["${data.aws_subnet.example.*.cidr_block}"]
}
```

## Argument Reference

* `vpc_id` - (Required) The VPC ID that you want to filter from.

## Attributes Reference

* `ids` - Is a list of all the subnet ids found. If none found. This data source will fail out.
