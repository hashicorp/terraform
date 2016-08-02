---
layout: "aws"
page_title: "AWS: aws_availability_zones"
sidebar_current: "docs-aws-datasource-availability-zones"
description: |-
    Provides a list of availability zones which can be used by an AWS account
---

# aws\_availability\_zones

The Availability Zones data source allows access to the list of AWS
Availability Zones which can be accessed by an AWS account within the region
configured in the provider.

## Example Usage

```
# Declare the data source
data "aws_availability_zones" "available" {}

# e.g. Create subnets in the first two available availability zones

resource "aws_subnet" "primary" {
  availability_zone = "${data.aws_availability_zones.available.names[0]}"

  # Other properties...
}

resource "aws_subnet" "secondary" {
  availability_zone = "${data.aws_availability_zones.available.names[1]}"

  # Other properties...
}
```

## Argument Reference

There are no arguments for this data source.

## Attributes Reference

The following attributes are exported:

* `names` - A list of the availability zone names available to the account.
