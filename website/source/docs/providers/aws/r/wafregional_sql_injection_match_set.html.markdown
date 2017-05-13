---
layout: "aws"
page_title: "AWS: wafregional_sql_injection_match_set"
sidebar_current: "docs-aws-resource-wafregional-sql-injection-match-set"
description: |-
  Provides a AWS WAF Regional SqlInjectionMatchSet resource for use with ALB.
---

# aws\_wafregional\_sql\_injection\_match\_set

Provides a WAF Regional SQL Injection Match Set Resource for use with Application Load Balancer.

## Example Usage

```
resource "aws_wafregional_sql_injection_match_set" "sql_injection_match_set" {
  name = "tf-sql_injection_match_set"
  sql_injection_match_tuples {
    text_transformation = "URL_DECODE"
    field_to_match {
      type = "QUERY_STRING"
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name or description of the SizeConstraintSet.
* `sql_injection_match_tuples` - The parts of web requests that you want AWS WAF to inspect for malicious SQL code and, if you want AWS WAF to inspect a header, the name of the header.

## Remarks

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the WAF SqlInjectionMatchSet.
