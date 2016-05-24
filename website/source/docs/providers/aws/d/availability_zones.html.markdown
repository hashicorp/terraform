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
data "aws_availability_zones" "zones" {}

# Create a subnet in each availability zone
resource "aws_subnet" "public" {
    count = "${length(data.aws_availability_zones.zones.instance)}"
    
    availability_zone = "${data.aws_availability_zones.zones.instance[count.index]}"

    # Other properties...
}
```

## Argument Reference

There are no arguments for this data source.

## Attributes Reference

The following attributes are exported:

* `instance` - A list of the availability zone names available to the account.
