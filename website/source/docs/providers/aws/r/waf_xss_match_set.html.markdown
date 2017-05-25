---
layout: "aws"
page_title: "AWS: waf_xss_match_set"
sidebar_current: "docs-aws-resource-waf-xss-match-set"
description: |-
  Provides a AWS WAF XssMatchSet resource.
---

# aws\_waf\_xss\_match\_set

Provides a WAF XSS Match Set Resource

## Example Usage

```hcl
resource "aws_waf_xss_match_set" "xss_match_set" {
  name = "xss_match_set"

  xss_match_tuples {
    text_transformation = "NONE"

    field_to_match {
      type = "URI"
    }
  }

  xss_match_tuples {
    text_transformation = "NONE"

    field_to_match {
      type = "QUERY_STRING"
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name or description of the SizeConstraintSet.
* `xss_match_tuples` - (Optional) The parts of web requests that you want to inspect for cross-site scripting attacks.

## Nested Blocks

### `xss_match_tuples`

* `field_to_match` - (Required) Specifies where in a web request to look for cross-site scripting attacks.
* `text_transformation` - (Required) Text transformations used to eliminate unusual formatting that attackers use in web requests in an effort to bypass AWS WAF.
  If you specify a transformation, AWS WAF performs the transformation on `target_string` before inspecting a request for a match.
  e.g. `CMD_LINE`, `HTML_ENTITY_DECODE` or `NONE`.
  See [docs](http://docs.aws.amazon.com/waf/latest/APIReference/API_XssMatchTuple.html#WAF-Type-XssMatchTuple-TextTransformation)
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

* `id` - The ID of the WAF XssMatchSet.
