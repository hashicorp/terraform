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
resource "aws_waf_web_acl" "waf_acl" {
  depends_on = ["aws_waf_ipset.ipset", "aws_waf_rule.wafrule"]
  name = "tfWebACL"
  metric_name = "tfWebACL"
  default_action {
    type = "ALLOW"
  }
  rules {
    action {
       type = "BLOCK"
    }
    priority = 1 
    rule_id = "${aws_waf_rule.wafrule.id}"
  }
}
```

## Argument Reference

The following arguments are supported:

* `default_action` - (Required) The action that you want AWS WAF to take when a request doesn't match the criteria in any of the rules that are associated with the web ACL.
* `metric_name` - (Required) The name or description for the Amazon CloudWatch metric of this web ACL.
* `name` - (Required) The name or description of the web ACL.
* `rules` - (Required) The rules to associate with the web ACL and the settings for each rule.


## Attributes Reference

The following attributes are exported:

* `id` - The ID of the WAF WebACL.
