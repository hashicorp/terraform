---
layout: "aws"
page_title: "AWS: aws_vpc_peering_connection_accept"
sidebar_current: "docs-aws-resource-vpc-peering-accept"
description: |-
  Provides a VPC Peering Connection Accept resource.
---

# aws\_vpc\_peering\_connection\_accept

Provides a VPC Peering Connection Accept resource.

## Example Usage

Basic usage:

```
resource "aws_vpc" "main" {
    cidr_block = "10.0.0.0/16"
}

provider "aws" {
    // another AWS account creds
    access_key = "..."
    secret_key = "..."
    alias = "peer"
}

resource "aws_vpc" "peer" {
    provider = "aws.peer"
    cidr_block = "10.1.0.0/16"
}

resource "aws_vpc_peering_connection" "peer" {
    vpc_id = "${aws_vpc.main.id}"
    peer_vpc_id = "${aws_vpc.peer.id}"
    auto_accept = false
}

resource "aws_vpc_peering_connection_accept" "peer" {
    provider = "aws.peer"
    peering_connection_id = "${aws_vpc_peering_connection.peer.id}"
}
```

## Argument Reference

The following arguments are supported:

* `peering_connection_id` - (Required) The VPC Peering Connection ID to accept.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the VPC Peering Connection.
* `accept_status` - The Status of the VPC peering connection request.
