---
layout: "aws"
page_title: "AWS: aws_route53_delegation_set"
sidebar_current: "docs-aws-resource-route53-delegation-set"
description: |-
  Provides a Route53 Delegation Set resource.
---

# aws\_route53\_delegation_set

Provides a [Route53 Delegation Set](https://docs.aws.amazon.com/Route53/latest/APIReference/actions-on-reusable-delegation-sets.html) resource.

## Example Usage

```
resource "aws_route53_delegation_set" "main" {
    reference_name = "DynDNS"
}

resource "aws_route53_zone" "primary" {
    name = "hashicorp.com"
    delegation_set_id = "${aws_route53_delegation_set.main.id}"
}

resource "aws_route53_zone" "secondary" {
    name = "terraform.io"
    delegation_set_id = "${aws_route53_delegation_set.main.id}"
}
```

## Argument Reference

The following arguments are supported:

* `reference_name` - (Optional) This is a reference name used in Caller Reference
  (helpful for identifying single delegation set amongst others)

## Attributes Reference

The following attributes are exported:

* `id` - The delegation set ID
* `name_servers` - A list of authoritative name servers for the hosted zone
  (effectively a list of NS records).
