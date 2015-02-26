---
layout: "aws"
page_title: "AWS: aws_route53_zone"
sidebar_current: "docs-aws-resource-route53-zone"
description: |-
  Provides a Route53 Hosted Zone resource.
---

# aws\_route53\_zone

Provides a Route53 Hosted Zone resource.

## Example Usage

```
resource "aws_route53_zone" "primary" {
   name = "example.com"
}
```

For use in subdomains, note that you need to create a
`aws_route53_record` of type `NS` as well as the subdomain
zone.

```
resource "aws_route53_zone" "main" {
  name = "example.com"
}

resource "aws_route53_zone" "dev" {
  name = "dev.example.com"
  parent_route53_zone = "${aws_route53_zone.main.zone_id}"
}

resource "aws_route53_record" "dev-ns" {
    zone_id = "${aws_route53_zone.main.zone_id}"
    name = "dev.example.com"
    type = "NS"
    ttl = "30"
    records = ["127.0.0.1", "127.0.0.27"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) This is the name of the hosted zone.

## Attributes Reference

The following attributes are exported:

* `zone_id` - The Hosted Zone ID. This can be referenced by zone records.

