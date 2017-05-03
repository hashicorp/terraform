---
layout: "aws"
page_title: "AWS: aws_waf_web_acl"
sidebar_current: "docs-aws-resource-waf-webacl"
description: |-
  Provides a AWS WAF web access control group (ACL) resource.
---

# aws\_waf\_web\_acl

Provides a WAF Web ACL Resource

## Example Usage

```hcl
resource "aws_waf_ipset" "ipset" {
  name = "tfIPSet"

  ip_set_descriptors {
    type  = "IPV4"
    value = "192.0.7.0/24"
  }
}

resource "aws_waf_rule" "wafrule" {
  depends_on  = ["aws_waf_ipset.ipset"]
  name        = "tfWAFRule"
  metric_name = "tfWAFRule"

  predicates {
    data_id = "${aws_waf_ipset.ipset.id}"
    negated = false
    type    = "IPMatch"
  }
}

resource "aws_waf_web_acl" "waf_acl" {
  depends_on  = ["aws_waf_ipset.ipset", "aws_waf_rule.wafrule"]
  name        = "tfWebACL"
  metric_name = "tfWebACL"

  default_action {
    type = "ALLOW"
  }

  rules {
    action {
      type = "BLOCK"
    }

    priority = 1
    rule_id  = "${aws_waf_rule.wafrule.id}"
  }
}
```

## Argument Reference

The following arguments are supported:

* `default_action` - (Required) The action that you want AWS WAF to take when a request doesn't match the criteria in any of the rules that are associated with the web ACL.
* `metric_name` - (Required) The name or description for the Amazon CloudWatch metric of this web ACL.
* `name` - (Required) The name or description of the web ACL.
* `rules` - (Required) The rules to associate with the web ACL and the settings for each rule.

## Nested Blocks

### `default_action`

#### Arguments

* `type` - (Required) Specifies how you want AWS WAF to respond to requests that match the settings in a rule.
  e.g. `ALLOW`, `BLOCK` or `COUNT`

### `rules`

See [docs](http://docs.aws.amazon.com/waf/latest/APIReference/API_ActivatedRule.html) for all details and supported values.

#### Arguments

* `action` - (Required) The action that CloudFront or AWS WAF takes when a web request matches the conditions in the rule.
  e.g. `ALLOW`, `BLOCK` or `COUNT`
* `priority` - (Required) Specifies the order in which the rules in a WebACL are evaluated.
  Rules with a lower value are evaluated before rules with a higher value.
* `rule_id` - (Required) ID of the associated [rule](/docs/providers/aws/r/waf_rule.html)

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the WAF WebACL.
