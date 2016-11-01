---
layout: "aws"
page_title: "AWS: waf_sql_injection_match_set"
sidebar_current: "docs-aws-resource-waf-sql-injection-match-set"
description: |-
  Provides a AWS WAF SqlInjectionMatchSet resource.
---

# aws\_waf\_sql\_injection\_match\_set

Provides a WAF SQL Injection Match Set Resource

## Example Usage

```
resource "aws_waf_sql_injection_match_set" "sql_injection_match_set" {
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
