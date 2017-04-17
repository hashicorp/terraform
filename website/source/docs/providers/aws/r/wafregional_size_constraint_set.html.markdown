---
layout: "aws"
page_title: "AWS: wafregional_size_constraint_set"
sidebar_current: "docs-aws-resource-wafregional-size-constraint-set"
description: |-
  Provides a AWS WAF Regional SizeConstraintSet resource for use with ALB.
---

# aws\_wafregional\_size\_constraint\_set

Provides a WAF Regional Size Constraint Set Resource for use with Application Load Balancer.

## Example Usage

```
resource "aws_wafregional_size_constraint_set" "size_constraint_set" {
  name = "tfsize_constraints"
  size_constraint {
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
* `size_constraint` - (Optional) The size constraint and the part of the web request to check.

SizeConstraint(size_constraint) support following:

* `name` - (Required) The name or description of the SizeConstraintSet.
* `field_to_match` - (Required) The name of the SizeConstraintSet.
* `text_transformation` - (Required) The formatting way for web request.
* `comparison_operator` - (Required) The type of comparison.
* `size` - (Required) The size of bytes.

FieldToMatch(field_to_match) support following:

* `data` - (Optional) When the value of Type is HEADER, enter the name of the header that you want AWS WAF to search, for example, User-Agent or Referer. If the value of Type is any other value, omit Data.
* `type` - (Required) The part of the web request that you want AWS WAF to search for a specified string.

## Remarks

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the WAF SizeConstraintSet.
