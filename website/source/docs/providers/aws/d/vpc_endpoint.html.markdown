---
layout: "aws"
page_title: "AWS: aws_vpc_endpoint"
sidebar_current: "docs-aws-datasource-vpc-endpoint-x"
description: |-
    Provides details about a specific VPC endpoint.
---

# aws\_vpc\_endpoint

The VPC Endpoint data source provides details about
a specific VPC endpoint.

## Example Usage

```hcl
# Declare the data source
data "aws_vpc_endpoint" "s3" {
  vpc_id       = "${aws_vpc.foo.id}"
  service_name = "com.amazonaws.us-west-2.s3"
}

resource "aws_vpc_endpoint_route_table_association" "private_s3" {
  vpc_endpoint_id = "${data.aws_vpc_endpoint.s3.id}"
  route_table_id  = "${aws_route_table.private.id}"
}
```

## Argument Reference

The arguments of this data source act as filters for querying the available VPC endpoints.
The given filters must match exactly one VPC endpoint whose data will be exported as attributes.

* `id` - (Optional) The ID of the specific VPC Endpoint to retrieve.

* `state` - (Optional) The state of the specific VPC Endpoint to retrieve.

* `vpc_id` - (Optional) The ID of the VPC in which the specific VPC Endpoint is used.

* `service_name` - (Optional) The AWS service name of the specific VPC Endpoint to retrieve.

## Attributes Reference

All of the argument attributes are also exported as result attributes.

* `policy` - The policy document associated with the VPC Endpoint.

* `route_table_ids` - One or more route tables associated with the VPC Endpoint.
