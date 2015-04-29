---
layout: "aws"
page_title: "AWS: aws_route53_record"
sidebar_current: "docs-aws-resource-route53-record"
description: |-
  Provides a Route53 record resource.
---

# aws\_route53\_record

Provides a Route53 record resource.

## Example Usage

### Simple routing policy

```
resource "aws_route53_record" "www" {
   zone_id = "${aws_route53_zone.primary.zone_id}"
   name = "www.example.com"
   type = "A"
   ttl = "300"
   records = ["${aws_eip.lb.public_ip}"]
}
```

### Weighted routing policy
See [AWS Route53 Developer Guide](http://docs.aws.amazon.com/Route53/latest/DeveloperGuide/routing-policy.html#routing-policy-weighted) for details.

```
resource "aws_route53_record" "www-dev" {
  zone_id = "${aws_route53_zone.primary.zone_id}"
  name = "www"
  type = "CNAME"
  ttl = "5"
  weight = 10
  set_identifier = "dev"
  records = ["dev.example.com"]
}

resource "aws_route53_record" "www-live" {
  zone_id = "${aws_route53_zone.primary.zone_id}"
  name = "www"
  type = "CNAME"
  ttl = "5"
  weight = 90
  set_identifier = "live"
  records = ["live.example.com"]
}
```

## Argument Reference

The following arguments are supported:

* `zone_id` - (Required) The ID of the hosted zone to contain this record.
* `name` - (Required) The name of the record.
* `type` - (Required) The record type.
* `ttl` - (Required) The TTL of the record.
* `records` - (Required) A string list of records.
* `weight` - (Optional) The weight of weighted record (0-255).
* `set_identifier` - (Optional) Unique identifier to differentiate weighted
  record from one another. Required for each weighted record.

## Attributes Reference

No attributes are exported.

