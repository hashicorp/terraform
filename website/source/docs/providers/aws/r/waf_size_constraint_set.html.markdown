---
layout: "aws"
page_title: "AWS: waf_size_constraint_set"
sidebar_current: "docs-aws-resource-waf-size-constraint-set"
description: |-
  Provides a AWS WAF Size Constraint Set resource.
---

# aws\_waf\_size\_constraint\_set

Provides a WAF Size Constraint Set Resource

## Example Usage

```hcl
resource "aws_waf_size_constraint_set" "size_constraint_set" {
  name = "tfsize_constraints"

  size_constraints {
    text_transformation = "NONE"
    comparison_operator = "EQ"
    size                = "4096"

    field_to_match {
      type = "BODY"
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name or description of the Size Constraint Set.
* `size_constraints` - (Optional) Specifies the parts of web requests that you want to inspect the size of.

## Nested Blocks

### `size_constraints`

#### Arguments

* `field_to_match` - (Required) Specifies where in a web request to look for the size constraint.
* `comparison_operator` - (Required) The type of comparison you want to perform.
  e.g. `EQ`, `NE`, `LT`, `GT`.
  See [docs](http://docs.aws.amazon.com/waf/latest/APIReference/API_SizeConstraint.html#WAF-Type-SizeConstraint-ComparisonOperator) for all supported values.
* `size` - (Required) The size in bytes that you want to compare against the size of the specified `field_to_match`.
  Valid values are between 0 - 21474836480 bytes (0 - 20 GB).
* `text_transformation` - (Required) Text transformations used to eliminate unusual formatting that attackers use in web requests in an effort to bypass AWS WAF.
  If you specify a transformation, AWS WAF performs the transformation on `field_to_match` before inspecting a request for a match.
  e.g. `CMD_LINE`, `HTML_ENTITY_DECODE` or `NONE`.
  See [docs](http://docs.aws.amazon.com/waf/latest/APIReference/API_SizeConstraint.html#WAF-Type-SizeConstraint-TextTransformation)
  for all supported values.
  **Note:** if you choose `BODY` as `type`, you must choose `NONE` because CloudFront forwards only the first 8192 bytes for inspection.

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

* `id` - The ID of the WAF Size Constraint Set.
