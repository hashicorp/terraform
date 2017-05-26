---
layout: "aws"
page_title: "AWS: aws_availability_zone"
sidebar_current: "docs-aws-datasource-availability-zone"
description: |-
    Provides details about a specific availability zone
---

# aws\_availability\_zone

`aws_availability_zone` provides details about a specific availability zone (AZ)
in the current region.

This can be used both to validate an availability zone given in a variable
and to split the AZ name into its component parts of an AWS region and an
AZ identifier letter. The latter may be useful e.g. for implementing a
consistent subnet numbering scheme across several regions by mapping both
the region and the subnet letter to network numbers.

This is different from the `aws_availability_zones` (plural) data source,
which provides a list of the available zones.

## Example Usage

The following example shows how this data source might be used to derive
VPC and subnet CIDR prefixes systematically for an availability zone.

```hcl
variable "region_number" {
  # Arbitrary mapping of region name to number to use in
  # a VPC's CIDR prefix.
  default = {
    us-east-1      = 1
    us-west-1      = 2
    us-west-2      = 3
    eu-central-1   = 4
    ap-northeast-1 = 5
  }
}

variable "az_number" {
  # Assign a number to each AZ letter used in our configuration
  default = {
    a = 1
    b = 2
    c = 3
    d = 4
    e = 5
    f = 6
  }
}

# Retrieve the AZ where we want to create network resources
# This must be in the region selected on the AWS provider.
data "aws_availability_zone" "example" {
  name = "eu-central-1a"
}

# Create a VPC for the region associated with the AZ
resource "aws_vpc" "example" {
  cidr_block = "${cidrsubnet("10.0.0.0/8", 4, var.region_number[data.aws_availability_zone.example.region])}"
}

# Create a subnet for the AZ within the regional VPC
resource "aws_subnet" "example" {
  vpc_id     = "${aws_vpc.example.id}"
  cidr_block = "${cidrsubnet(aws_vpc.example.cidr_block, 4, var.az_number[data.aws_availability_zone.example.name_suffix])}"
}
```

## Argument Reference

The arguments of this data source act as filters for querying the available
availability zones. The given filters must match exactly one availability
zone whose data will be exported as attributes.

* `name` - (Optional) The full name of the availability zone to select.

* `state` - (Optional) A specific availability zone state to require. May
  be any of `"available"`, `"information"`, `"impaired"` or `"available"`.

All reasonable uses of this data source will specify `name`, since `state`
alone would match a single AZ only in a region that itself has only one AZ.

## Attributes Reference

The following attributes are exported:

* `name` - The name of the selected availability zone.

* `region` - The region where the selected availability zone resides.
  This is always the region selected on the provider, since this data source
  searches only within that region.

* `name_suffix` - The part of the AZ name that appears after the region name,
  uniquely identifying the AZ within its region.

* `state` - The current state of the AZ.
