---
layout: "aws"
page_title: "AWS: wafregional_xss_match_set"
sidebar_current: "docs-aws-resource-wafregional-xss-match-set"
description: |-
  Provides a AWS WAF Regional XssMatchSet resource for use with ALB.
---

# aws\_wafregional\_xss\_match\_set

Provides a WAF Regional XSS Match Set Resource for use with Application Load Balancer.

## Example Usage

```
resource "aws_wafregional_xss_match_set" "xss_match_set" {
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
* `xss_match_tuples` - The parts of web requests that you want to inspect for cross-site scripting attacks.

## Remarks

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the WAF XssMatchSet.
