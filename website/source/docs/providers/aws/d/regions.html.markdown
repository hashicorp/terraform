---
layout: "aws"
page_title: "AWS: aws_regions"
sidebar_current: "docs-aws-datasource-regions"
description: |-
    Provides a list of AWS regions.
---

# aws\_regions

Provides a list of all AWS regions. See [AWS Regions and Endpoints][1].

This is different from the `aws_region` (singular) data source, which provides
some details about a specific region.

## Example Usage

```hcl
# Declare the data source
data "aws_regions" "regions" {}

## Argument Reference

There are no arguments available for this data source.

## Attributes Reference

The following attributes are exported:

* `names` - A list of the regions.

[1]: http://docs.aws.amazon.com/general/latest/gr/rande.html
