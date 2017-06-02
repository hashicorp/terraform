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

```hcl
resource "aws_route53_record" "www" {
  zone_id = "${aws_route53_zone.primary.zone_id}"
  name    = "www.example.com"
  type    = "A"
  ttl     = "300"
  records = ["${aws_eip.lb.public_ip}"]
}
```

### Weighted routing policy
Other routing policies are configured similarly. See [AWS Route53 Developer Guide](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/routing-policy.html) for details.

```hcl
resource "aws_route53_record" "www-dev" {
  zone_id = "${aws_route53_zone.primary.zone_id}"
  name    = "www"
  type    = "CNAME"
  ttl     = "5"

  weighted_routing_policy {
    weight = 10
  }

  set_identifier = "dev"
  records        = ["dev.example.com"]
}

resource "aws_route53_record" "www-live" {
  zone_id = "${aws_route53_zone.primary.zone_id}"
  name    = "www"
  type    = "CNAME"
  ttl     = "5"

  weighted_routing_policy {
    weight = 90
  }

  set_identifier = "live"
  records        = ["live.example.com"]
}
```

### Alias record
See [related part of AWS Route53 Developer Guide](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/resource-record-sets-choosing-alias-non-alias.html)
to understand differences between alias and non-alias records.

TTL for all alias records is [60 seconds](https://aws.amazon.com/route53/faqs/#dns_failover_do_i_need_to_adjust),
you cannot change this, therefore `ttl` has to be omitted in alias records.

```hcl
resource "aws_elb" "main" {
  name               = "foobar-terraform-elb"
  availability_zones = ["us-east-1c"]

  listener {
    instance_port     = 80
    instance_protocol = "http"
    lb_port           = 80
    lb_protocol       = "http"
  }
}

resource "aws_route53_record" "www" {
  zone_id = "${aws_route53_zone.primary.zone_id}"
  name    = "example.com"
  type    = "A"

  alias {
    name                   = "${aws_elb.main.dns_name}"
    zone_id                = "${aws_elb.main.zone_id}"
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
* `set_identifier` - (Optional) Unique identifier to differentiate records with routing policies from one another. Required if using `failover`, `geolocation`, `latency`, or `weighted` routing policies documented below.
* `health_check_id` - (Optional) The health check the record should be associated with.
* `alias` - (Optional) An alias block. Conflicts with `ttl` & `records`.
  Alias record documented below.
* `failover_routing_policy` - (Optional) A block indicating the routing behavior when associated health check fails. Conflicts with any other routing policy. Documented below.
* `geolocation_routing_policy` - (Optional) A block indicating a routing policy based on the geolocation of the requestor. Conflicts with any other routing policy. Documented below.
* `latency_routing_policy` - (Optional) A block indicating a routing policy based on the latency between the requestor and an AWS region. Conflicts with any other routing policy. Documented below.
* `weighted_routing_policy` - (Optional) A block indicating a weighted routing policy. Conflicts with any other routing policy. Documented below.

Exactly one of `records` or `alias` must be specified: this determines whether it's an alias record.

Alias records support the following:

* `name` - (Required) DNS domain name for a CloudFront distribution, S3 bucket, ELB, or another resource record set in this hosted zone.
* `zone_id` - (Required) Hosted zone ID for a CloudFront distribution, S3 bucket, ELB, or Route 53 hosted zone. See [`resource_elb.zone_id`](/docs/providers/aws/r/elb.html#zone_id) for example.
* `evaluate_target_health` - (Required) Set to `true` if you want Route 53 to determine whether to respond to DNS queries using this resource record set by checking the health of the resource record set. Some resources have special requirements, see [related part of documentation](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/resource-record-sets-values.html#rrsets-values-alias-evaluate-target-health).

Failover routing policies support the following:

* `type` - (Required) `PRIMARY` or `SECONDARY`. A `PRIMARY` record will be served if its healthcheck is passing, otherwise the `SECONDARY` will be served. See http://docs.aws.amazon.com/Route53/latest/DeveloperGuide/dns-failover-configuring-options.html#dns-failover-failover-rrsets

Geolocation routing policies support the following:

* `continent` - A two-letter continent code. See http://docs.aws.amazon.com/Route53/latest/APIReference/API_GetGeoLocation.html for code details. Either `continent` or `country` must be specified.
* `country` - A two-character country code or `*` to indicate a default resource record set.
* `subdivision` - (Optional) A subdivision code for a country.

Latency routing policies support the following:

* `region` - (Required) An AWS region from which to measure latency. See http://docs.aws.amazon.com/Route53/latest/DeveloperGuide/routing-policy.html#routing-policy-latency

Weighted routing policies support the following:

* `weight` - (Required) A numeric value indicating the relative weight of the record. See http://docs.aws.amazon.com/Route53/latest/DeveloperGuide/routing-policy.html#routing-policy-weighted.

## Attributes Reference

* `fqdn` - [FQDN](https://en.wikipedia.org/wiki/Fully_qualified_domain_name) built using the zone domain and `name`


## Import

Route53 Records can be imported using ID of the record. The ID is made up as ZONEID_RECORDNAME_TYPE_SET-IDENTIFIER

e.g.

```
Z4KAPRWWNC7JR_dev.example.com_NS_dev
```

In this example, `Z4KAPRWWNC7JR` is the ZoneID, `dev.example.com` is the Record Name, `NS` is the Type and `dev` is the Set Identifier.
Only the Set Identifier is actually optional in the ID

To import the ID above, it would look as follows:

```
$ terraform import aws_route53_record.myrecord Z4KAPRWWNC7JR_dev.example.com_NS_dev
```
