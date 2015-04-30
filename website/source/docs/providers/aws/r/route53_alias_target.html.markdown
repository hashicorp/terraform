---
layout: "aws"
page_title: "AWS: aws_route53_alias_target"
sidebar_current: "docs-aws-resource-route53-alias_target"
description: |-
  Provides a Route53 alias_target resource.
---

# aws\_route53\_alias_target

Provides a Route53 alias target resource.

## Example Usage

```
resource "aws_route53_alias_target" "www" {
   zone_id = "${aws_route53_zone.primary.zone_id}"
   name = "www.example.com"
   type = "A"
   target = ["${aws_elb.default.dns_name}"]
   evaluate_health = false
}
```

## Argument Reference

The following arguments are supported:

* `zone_id` - (Required) The ID of the hosted zone to contain this record.
* `name` - (Required) The name of the record.
* `type` - (Required) The record type.
* `target` - (Required) The DNS name to alias.
* `target_zone_id` - (Required) The `zone_id` of the target. `zone_id` is accessible on `aws_route53_record` and `aws_elb` resources.
* `evaluate_health` - (Optional) Determine if the health of the target is evaluated. Defaults to false.

## Attributes Reference

No attributes are exported.

