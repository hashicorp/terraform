---
layout: "aws"
page_title: "AWS: aws_wafregional_web_acl_association"
sidebar_current: "docs-aws-resource-wafregional-web-acl-association"
description: |-
  Provides a resource to create an association between a WAF Regional WebACL and Application Load Balancer.
---

# aws\_wafregional\_web\_acl\_association

Provides a resource to create an association between a WAF Regional WebACL and Application Load Balancer.

-> **Note:** An Application Load Balancer can only be associated with one WAF Regional WebACL.

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

resource "aws_wafregional_web_acl" "wafacl" {
  depends_on = ["aws_wafregional_ipset.ipset", "aws_wafregional_rule.wafrule"]
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
    rule_id = "${aws_wafregional_rule.wafrule.id}"
  }
}

resource "aws_vpc" "main" {
	cidr_block = "10.1.0.0/16"
}

resource "aws_subnet" "foo" {
	vpc_id = "${aws_vpc.main.id}"
	cidr_block = "10.1.1.0/24"
}

resource "aws_subnet" "bar" {
	vpc_id = "${aws_vpc.main.id}"
	cidr_block = "10.1.2.0/24"
}

resource "aws_alb" "alb" {
    subnets = ["${aws_subnet.foo.id}", "${aws_subnet.bar.id}"]
}

resource "aws_wafregional_web_acl_association" "wafassociation" {
    depends_on = ["aws_alb.alb", "aws_wafregional_web_acl.wafacl"]
    web_acl_id = "${aws_wafregional_web_acl.wafacl.id}"
    resource_arn = "${aws_alb.alb.arn}"
}
```

## Argument Reference

The following arguments are supported:

* `web_acl_id` - (Required) The ID of the WAF Regional WebACL to create an association.
* `resource_arn` - (Required) Application Load Balancer ARN to associate with.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the association
