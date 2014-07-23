---
layout: "aws"
page_title: "AWS: aws_route53_zone"
sidebar_current: "docs-aws-resource-route53-zone"
---

# aws\_route53\_zone

Provides a Route53 Hosted Zone resource.

## Example Usage

```
resource "aws_route53_zone" "primary" {
   name = "example.com"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) This is the name of the hosted zone.

## Attributes Reference

The following attributes are exported:

* `zone_id` - The Hosted Zone ID. This can be referenced by zone records.

