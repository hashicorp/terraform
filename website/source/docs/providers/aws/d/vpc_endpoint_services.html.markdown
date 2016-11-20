---
layout: "aws"
page_title: "AWS: aws_vpc_endpoint_services"
sidebar_current: "docs-aws-datasource-vpc-endpoint-services"
description: |-
    Provides a list of all supported AWS services that can be specified when creating a VPC endpoint.
---

# aws\_vpc\_endpoint\_services

The VPC Endpoint Services data source allows access to the list of AWS
services that can be specified when creating a VPC endpoint within the region
configured in the provider.

## Example Usage

```
# Declare the data source
data "aws_vpc_endpoint_services" "es" {}

# Create a VPC
resource "aws_vpc" "foo" {
    cidr_block = "10.0.0.0/16"
}

# Create a VPC endpoint
resource "aws_vpc_endpoint" "ep" {
    vpc_id = "${aws_vpc.foo.id}"
    service_name = "${data.aws_vpc_endpoint_services.es.names[0]}"
}
```

## Argument Reference

There are no arguments available for this data source.

## Attributes Reference

The following attributes are exported:

* `names` - A list of the AWS services that can be specified when creating a VPC endpoint.
