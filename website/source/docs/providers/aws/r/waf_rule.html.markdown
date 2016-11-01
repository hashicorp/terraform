---
layout: "aws"
page_title: "AWS: waf_rule"
sidebar_current: "docs-aws-resource-waf-rule"
description: |-
  Provides a AWS WAF rule resource.
---

# aws\_waf\_rule

Provides a WAF Rule Resource

## Example Usage

```
resource "aws_waf_ipset" "ipset" {
  name = "tfIPSet"
  ip_set_descriptors {
    type = "IPV4"
    value = "192.0.7.0/24"
  }
}

resource "aws_waf_rule" "wafrule" {
  depends_on = ["aws_waf_ipset.ipset"]
  name = "tfWAFRule"
  metric_name = "tfWAFRule"
  predicates {
    data_id = "${aws_waf_ipset.ipset.id}"
    negated = false
    type = "IPMatch"
  }
}
```

## Argument Reference

The following arguments are supported:

* `metric_name` - (Required) The name or description for the Amazon CloudWatch metric of this rule.
* `name` - (Required) The name or description of the rule.
* `predicates` - (Optional) The ByteMatchSet, IPSet, SizeConstraintSet, SqlInjectionMatchSet, or XssMatchSet objects to include in a rule.

## Remarks

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the WAF rule.
