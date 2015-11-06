---
layout: "aws"
page_title: "AWS: aws_vpc_endpoint"
sidebar_current: "docs-aws-resource-vpc-endpoint"
description: |-
  Provides a VPC Endpoint resource.
---

# aws\_vpc\_endpoint

Provides a VPC Endpoint resource.

## Example Usage

Basic usage:

```
resource "aws_vpc_endpoint" "private-s3" {
    vpc_id = "${aws_vpc.main.id}"
    service_name = "com.amazonaws.us-west-2.s3"
}
```

## Argument Reference

The following arguments are supported:

* `vpc_id` - (Required) The ID of the VPC in which the endpoint will be used.
* `service_name` - (Required) The AWS service name, in the form `com.amazonaws.region.service`.
* `policy` - (Optional) A policy to attach to the endpoint that controls access to the service.
* `route_table_ids` - (Optional) One or more route table IDs.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the VPC endpoint.
