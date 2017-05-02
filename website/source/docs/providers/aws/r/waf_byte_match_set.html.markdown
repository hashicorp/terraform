---
layout: "aws"
page_title: "AWS: waf_byte_match_set"
sidebar_current: "docs-aws-resource-waf-bytematchset"
description: |-
  Provides a AWS WAF Byte Match Set resource.
---

# aws\_waf\_byte\_match\_set

Provides a WAF Byte Match Set Resource

## Example Usage

```hcl
resource "aws_waf_byte_match_set" "byte_set" {
  name = "tf_waf_byte_match_set"

  byte_match_tuples {
    text_transformation   = "NONE"
    target_string         = "badrefer1"
    positional_constraint = "CONTAINS"

    field_to_match {
      type = "HEADER"
      data = "referer"
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name or description of the Byte Match Set.
* `byte_match_tuples` - Specifies the bytes (typically a string that corresponds
  with ASCII characters) that you want to search for in web requests,
  the location in requests that you want to search, and other settings.

## Nested blocks

### `byte_match_tuples`

#### Arguments

* `field_to_match` - (Required) The part of a web request that you want to search, such as a specified header or a query string.
* `positional_constraint` - (Required) Within the portion of a web request that you want to search
  (for example, in the query string, if any), specify where you want to search.
  e.g. `CONTAINS`, `CONTAINS_WORD` or `EXACTLY`.
  See [docs](http://docs.aws.amazon.com/waf/latest/APIReference/API_ByteMatchTuple.html#WAF-Type-ByteMatchTuple-PositionalConstraint)
  for all supported values.
* `target_string` - (Optional) The value that you want to search for. e.g. `HEADER`, `METHOD` or `BODY`.
  See [docs](http://docs.aws.amazon.com/waf/latest/APIReference/API_ByteMatchTuple.html#WAF-Type-ByteMatchTuple-TargetString)
  for all supported values.
* `text_transformation` - (Required) Text transformations used to eliminate unusual formatting that attackers use in web requests in an effort to bypass AWS WAF.
  If you specify a transformation, AWS WAF performs the transformation on `target_string` before inspecting a request for a match.
  e.g. `CMD_LINE`, `HTML_ENTITY_DECODE` or `NONE`.
  See [docs](http://docs.aws.amazon.com/waf/latest/APIReference/API_ByteMatchTuple.html#WAF-Type-ByteMatchTuple-TextTransformation)
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

* `id` - The ID of the WAF Byte Match Set.
