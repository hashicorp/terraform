---
layout: "aws"
page_title: "AWS: waf_sql_injection_match_set"
sidebar_current: "docs-aws-resource-waf-sql-injection-match-set"
description: |-
  Provides a AWS WAF SQL Injection Match Set resource.
---

# aws\_waf\_sql\_injection\_match\_set

Provides a WAF SQL Injection Match Set Resource

## Example Usage

```hcl
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
* `sql_injection_match_tuples` - (Optional) The parts of web requests that you want AWS WAF to inspect for malicious SQL code and, if you want AWS WAF to inspect a header, the name of the header.

## Nested Blocks

### `sql_injection_match_tuples`

* `field_to_match` - (Required) Specifies where in a web request to look for snippets of malicious SQL code.
* `text_transformation` - (Required) Text transformations used to eliminate unusual formatting that attackers use in web requests in an effort to bypass AWS WAF.
  If you specify a transformation, AWS WAF performs the transformation on `field_to_match` before inspecting a request for a match.
  e.g. `CMD_LINE`, `HTML_ENTITY_DECODE` or `NONE`.
  See [docs](http://docs.aws.amazon.com/waf/latest/APIReference/API_SqlInjectionMatchTuple.html#WAF-Type-SqlInjectionMatchTuple-TextTransformation)
  for all supported values.

### `field_to_match`

#### Arguments

* `data` - (Optional) When `type` is `HEADER`, enter the name of the header that you want to search, e.g. `User-Agent` or `Referer`.
  If `type` is any other value, omit this field.
* `type` - (Required) The part of the web request that you want AWS WAF to search for a specified string.
  e.g. `HEADER`, `METHOD` or `BODY`.
  See [docs](http://docs.aws.amazon.com/waf/latest/APIReference/API_FieldToMatch.html)
  for all supported values.


## Remarks

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the WAF SQL Injection Match Set.
