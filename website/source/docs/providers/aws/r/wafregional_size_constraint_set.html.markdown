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
