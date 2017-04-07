---
layout: "aws"
page_title: "AWS: aws_route53_zone"
sidebar_current: "docs-aws-datasource-route53-zone"
description: |-
    Provides details about a specific Route 53 Hosted Zone
---

# aws\_route53\_zone

`aws_route53_zone` provides details about a specific Route 53 Hosted Zone.

This data source allows to find a Hosted Zone ID given Hosted Zone name and certain search criteria.

## Example Usage

The following example shows how to get a Hosted Zone from it's name and from this data how to create a Record Set.


```hcl
data "aws_route53_zone" "selected" {
  name         = "test.com."
  private_zone = true
}

resource "aws_route53_record" "www" {
  zone_id = "${data.aws_route53_zone.selected.zone_id}"
  name    = "www.${data.aws_route53_zone.selected.name}"
  type    = "A"
  ttl     = "300"
  records = ["10.0.0.1"]
}
```

## Argument Reference

The arguments of this data source act as filters for querying the available
Hosted Zone. You have to use `zone_id` or `name`, not both of them. The given filter must match exactly one
Hosted Zone. If you use `name` field for private Hosted Zone, you need to add `private_zone` field to `true`

* `zone_id` - (Optional) The Hosted Zone id of the desired Hosted Zone.

* `name` - (Optional) The Hosted Zone name of the desired Hosted Zone.
* `private_zone` - (Optional) Used with `name` field to get a private Hosted Zone.
* `vpc_id` - (Optional) Used with `name` field to get a private Hosted Zone associated with the vpc_id (in this case, private_zone is not mandatory).
* `tags` - (Optional) Used with `name` field. A mapping of tags, each pair of which must exactly match
a pair on the desired security group.
## Attributes Reference

All of the argument attributes are also exported as
result attributes. This data source will complete the data by populating
any fields that are not included in the configuration with the data for
the selected Hosted Zone.

The following attribute is additionally exported:

* `caller_reference` - Caller Reference of the Hosted Zone.
* `comment` - The comment field of the Hosted Zone.
* `resource_record_set_count` - the number of Record Set in the Hosted Zone
