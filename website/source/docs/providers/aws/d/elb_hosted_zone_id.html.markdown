---
layout: "aws"
page_title: "AWS: aws_elb_hosted_zone_id"
sidebar_current: "docs-aws-datasource-elb-hosted-zone-id"
description: |-
  Get AWS Elastic Load Balancing Hosted Zone Id
---

# aws\_elb\_hosted\_zone\_id

Use this data source to get the HostedZoneId of the AWS Elastic Load Balancing HostedZoneId
in a given region for the purpose of using in an AWS Route53 Alias.

## Example Usage

```hcl
data "aws_elb_hosted_zone_id" "main" {}

resource "aws_route53_record" "www" {
  zone_id = "${aws_route53_zone.primary.zone_id}"
  name    = "example.com"
  type    = "A"

  alias {
    name                   = "${aws_elb.main.dns_name}"
    zone_id                = "${data.aws_elb_hosted_zone_id.main.id}"
    evaluate_target_health = true
  }
}
```

## Argument Reference

* `region` - (Optional) Name of the region whose AWS ELB HostedZoneId is desired.
  Defaults to the region from the AWS provider configuration.


## Attributes Reference

* `id` - The ID of the AWS ELB HostedZoneId in the selected region.
