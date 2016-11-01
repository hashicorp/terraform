---
layout: "aws"
page_title: "AWS: waf_size_constraint_set"
sidebar_current: "docs-aws-resource-waf-size-constraint-set"
description: |-
  Provides a AWS WAF SizeConstraintSet resource.
---

# aws\_waf\_size\_constraint\_set

Provides a WAF Size Constraint Set Resource

## Example Usage

```
resource "aws_waf_size_constraint_set" "size_constraint_set" {
  name = "tfsize_constraints"
  size_constraints {
    text_transformation = "NONE"
    comparison_operator = "EQ"
    size = "4096"
    field_to_match {
      type = "BODY"
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name or description of the SizeConstraintSet.
* `size_constraints` - (Required) The size constraint and the part of the web request to check.

## Remarks

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the WAF SizeConstraintSet.
