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

  tags {
    Environment = "dev"
  }
}

resource "aws_route53_record" "dev-ns" {
    zone_id = "${aws_route53_zone.main.zone_id}"
    name = "dev.example.com"
    type = "NS"
    ttl = "30"
    records = [
        "${aws_route53_zone.dev.name_servers.0}",
        "${aws_route53_zone.dev.name_servers.1}",
        "${aws_route53_zone.dev.name_servers.2}",
        "${aws_route53_zone.dev.name_servers.3}"
    ]
}
```

~> **NOTE:** The `name_servers` set is populated only for public Hosted Zones.
Private Zones will contain any empty set since AWS does not return a `DelegationSet`
for private Hosted Zones.

## Argument Reference

The following arguments are supported:

* `name` - (Required) This is the name of the hosted zone.
* `tags` - (Optional) A mapping of tags to assign to the zone.
* `vpc_id` - (Optional) The VPC to associate with a private hosted zone. Specifying `vpc_id` will create a private hosted zone.
* `vpc_region` - (Optional) The VPC's region. Defaults to the region of the AWS provider.

## Attributes Reference

The following attributes are exported:

* `zone_id` - The Hosted Zone ID. This can be referenced by zone records.
* `name_servers` - A list of name servers in a default delegation set. Support only for Public Hosted Zones.
  Find more about delegation sets in [AWS docs](http://docs.aws.amazon.com/Route53/latest/APIReference/actions-on-reusable-delegation-sets.html).
