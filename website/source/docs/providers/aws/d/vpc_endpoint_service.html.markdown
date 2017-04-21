---
layout: "aws"
page_title: "AWS: aws_vpc_endpoint_service"
sidebar_current: "docs-aws-datasource-vpc-endpoint-service"
description: |-
    Provides details about a specific AWS service that can be specified when creating a VPC endpoint.
---

# aws\_vpc\_endpoint\_service

The VPC Endpoint Service data source allows access to a specific AWS
service that can be specified when creating a VPC endpoint within the region
configured in the provider.

## Example Usage

```hcl
# Declare the data source
data "aws_vpc_endpoint_service" "s3" {
  service = "s3"
}

# Create a VPC
resource "aws_vpc" "foo" {
  cidr_block = "10.0.0.0/16"
}

# Create a VPC endpoint
resource "aws_vpc_endpoint" "ep" {
  vpc_id       = "${aws_vpc.foo.id}"
  service_name = "${data.aws_vpc_endpoint_service.s3.service_name}"
}
```

## Argument Reference

The arguments of this data source act as filters for querying the available VPC endpoint services.
The given filters must match exactly one VPC endpoint service whose data will be exported as attributes.

* `service` - (Required) The common name of the AWS service (e.g. `s3`).

## Attributes Reference

The following attributes are exported:

* `service_name` - The service name of the AWS service that can be specified when creating a VPC endpoint.
