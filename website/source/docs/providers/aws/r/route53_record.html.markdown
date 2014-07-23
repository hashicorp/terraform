---
layout: "aws"
page_title: "AWS: aws_route53_record"
sidebar_current: "docs-aws-resource-route53-record"
---

# aws\_route53\_record

Provides a Route53 record resource.

## Example Usage

```
resource "aws_route53_record" "www" {
   zone_id = "${aws_route53_zone.primary.zone_id}"
   name = "www.example.com"
   type = "A"
   ttl = "300"
   records = ["${aws_eip.lb.public_ip}"]
}
```

## Argument Reference

The following arguments are supported:

* `zone_id` - (Required) The ID of the hosted zone to contain this record.
* `name` - (Required) The name of the record.
* `type` - (Required) The record type.
* `ttl` - (Required) The TTL of the record.
* `records` - (Required) A string list of records.

## Attributes Reference

No attributes are exported.

