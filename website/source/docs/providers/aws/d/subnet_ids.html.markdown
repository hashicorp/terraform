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
  id = "${data.aws_subnet_ids.example.ids[count.index]}"
}

output "subnet_cidr_blocks" {
  value = ["${data.aws_subnet.example.*.cidr_block}"]
}
```

The following example retrieves a list of all subnets in a VPC with a custom
tag of `Tier` set to a value of "Private" so that the `aws_instance` resource
can loop through the subnets, putting instances across availability zones.

```hcl
data "aws_subnet_ids" "private" {
  vpc_id = "${var.vpc_id}"
  tags {
    Tier = "Private"
  }
}

resource "aws_instance" "app" {
  count         = "3"
  ami           = "${var.ami}"
  instance_type = "t2.micro"
  subnet_id     = "${element(data.aws_subnet_ids.private.ids, count.index)}"
}
```

## Argument Reference

* `vpc_id` - (Required) The VPC ID that you want to filter from.

* `tags` - (Optional) A mapping of tags, each pair of which must exactly match
  a pair on the desired subnets.

## Attributes Reference

* `ids` - Is a list of all the subnet ids found. If none found. This data source will fail out.
