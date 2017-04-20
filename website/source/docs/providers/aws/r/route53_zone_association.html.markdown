---
layout: "aws"
page_title: "AWS: aws_route53_zone_association"
sidebar_current: "docs-aws-resource-route53-zone-association"
description: |-
  Provides a Route53 private Hosted Zone to VPC association resource.
---

# aws\_route53\_zone\_association

Provides a Route53 private Hosted Zone to VPC association resource.

## Example Usage

```hcl
resource "aws_vpc" "primary" {
  cidr_block           = "10.6.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true
}

resource "aws_vpc" "secondary" {
  cidr_block           = "10.7.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true
}

resource "aws_route53_zone" "example" {
  name   = "example.com"
  vpc_id = "${aws_vpc.primary.id}"
}

resource "aws_route53_zone_association" "secondary" {
  zone_id = "${aws_route53_zone.example.zone_id}"
  vpc_id  = "${aws_vpc.secondary.id}"
}
```

## Argument Reference

The following arguments are supported:

* `zone_id` - (Required) The private hosted zone to associate.
* `vpc_id` - (Required) The VPC to associate with the private hosted zone.
* `vpc_region` - (Optional) The VPC's region. Defaults to the region of the AWS provider.

## Attributes Reference

The following attributes are exported:

* `id` - The calculated unique identifier for the association.
* `zone_id` - The ID of the hosted zone for the association.
* `vpc_id` - The ID of the VPC for the association.
* `vpc_region` - The region in which the VPC identified by `vpc_id` was created.
