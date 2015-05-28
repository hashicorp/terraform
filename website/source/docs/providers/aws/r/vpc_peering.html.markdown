---
layout: "aws"
page_title: "AWS: aws_vpc_peering_connection"
sidebar_current: "docs-aws-resource-vpc-peering"
description: |-
  Provides an VPC Peering Connection resource.
---

# aws\_vpc\_peering\_connection

Provides an VPC Peering Connection resource.

## Example Usage

Basic usage:

```
resource "aws_vpc" "main" {
    cidr_block = "10.0.0.0/16"
}
```

Basic usage with tags:

```

resource "aws_vpc_peering_connection" "foo" {
    peer_owner_id = "${var.peer_owner_id}"
    peer_vpc_id = "${aws_vpc.bar.id}"
    vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_vpc" "foo" {
    cidr_block = "10.1.0.0/16"
}

resource "aws_vpc" "bar" {
    cidr_block = "10.2.0.0/16"
}
```

## Argument Reference

The following arguments are supported:

* `peer_owner_id` - (Required) The AWS account ID of the owner of the peer VPC.
* `peer_vpc_id` - (Required) The ID of the VPC with which you are creating the VPC peering connection.
* `vpc_id` - (Required) The ID of the requester VPC.
* `auto_accept` - (Optional) Accept the peering (you need to be the owner of both VPCs).
* `tags` - (Optional) A mapping of tags to assign to the resource.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the VPC Peering Connections
* `accept_status` - The Status of the VPC peering connection request.


## Notes
You still have to accept the peering with the AWS Console, aws-cli or aws-sdk-go.
