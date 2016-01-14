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

### Alias record
See [related part of AWS Route53 Developer Guide](http://docs.aws.amazon.com/Route53/latest/DeveloperGuide/resource-record-sets-choosing-alias-non-alias.html)
to understand differences between alias and non-alias records.

TTL for all alias records is [60 seconds](http://aws.amazon.com/route53/faqs/#dns_failover_do_i_need_to_adjust),
you cannot change this, therefore `ttl` has to be omitted in alias records.

```
resource "aws_elb" "main" {
  name = "foobar-terraform-elb"
  availability_zones = ["us-east-1c"]

  listener {
    instance_port = 80
    instance_protocol = "http"
    lb_port = 80
    lb_protocol = "http"
  }
}

resource "aws_route53_record" "www" {
  zone_id = "${aws_route53_zone.primary.zone_id}"
  name = "example.com"
  type = "A"

  alias {
    name = "${aws_elb.main.dns_name}"
    zone_id = "${aws_elb.main.zone_id}"
    evaluate_target_health = true
  }
}
```

## Argument Reference

The following arguments are supported:

* `zone_id` - (Required) The ID of the hosted zone to contain this record.
* `name` - (Required) The name of the record.
* `type` - (Required) The record type.
* `ttl` - (Required for non-alias records) The TTL of the record.
* `records` - (Required for non-alias records) A string list of records.
* `weight` - (Optional) The weight of weighted record (0-255).
* `set_identifier` - (Optional) Unique identifier to differentiate weighted
record from one another. Required for each weighted record.
* `failover` - (Optional) The routing behavior when associated health check fails. Must be PRIMARY or SECONDARY.
* `health_check_id` - (Optional) The health check the record should be associated with.
* `alias` - (Optional) An alias block. Conflicts with `ttl` & `records`.
  Alias record documented below.

~> **Note:** The `weight` attribute uses a special sentinel value of `-1` for a
default in Terraform. This allows Terraform to distinquish between a `0` value
and an empty value in the configuration (none specified). As a result, a 
`weight` of `-1` will be present in the statefile if `weight` is omitted in the 
configuration.

Exactly one of `records` or `alias` must be specified: this determines whether it's an alias record.

Alias records support the following:

* `name` - (Required) DNS domain name for a CloudFront distribution, S3 bucket, ELB, or another resource record set in this hosted zone.
* `zone_id` - (Required) Hosted zone ID for a CloudFront distribution, S3 bucket, ELB, or Route 53 hosted zone. See [`resource_elb.zone_id`](/docs/providers/aws/r/elb.html#zone_id) for example.
* `evaluate_target_health` - (Required) Set to `true` if you want Route 53 to determine whether to respond to DNS queries using this resource record set by checking the health of the resource record set. Some resources have special requirements, see [related part of documentation](http://docs.aws.amazon.com/Route53/latest/DeveloperGuide/resource-record-sets-values.html#rrsets-values-alias-evaluate-target-health).

## Attributes Reference

* `fqdn` - [FQDN](http://en.wikipedia.org/wiki/Fully_qualified_domain_name) built using the zone domain and `name`

