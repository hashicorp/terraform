---
layout: "aws"
page_title: "AWS: wafregional_rule"
sidebar_current: "docs-aws-resource-wafregional-rule"
description: |-
  Provides a AWS WAF Regional rule resource for use with ALB.
---

# aws\_wafregional\_rule

Provides a WAF Regional Rule Resource for use with Application Load Balancer.

## Example Usage

```
resource "aws_wafregional_ipset" "ipset" {
  name = "tfIPSet"
  ip_set_descriptors {
    type = "IPV4"
    value = "192.0.7.0/24"
  }
}

resource "aws_wafregional_rule" "wafrule" {
  depends_on = ["aws_wafregional_ipset.ipset"]
  name = "tfWAFRule"
  metric_name = "tfWAFRule"
  predicates {
    data_id = "${aws_wafregional_ipset.ipset.id}"
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
