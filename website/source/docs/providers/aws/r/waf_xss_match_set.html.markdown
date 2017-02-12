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

```
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
* `xss_match_tuples` - The parts of web requests that you want to inspect for cross-site scripting attacks.

## Remarks

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the WAF XssMatchSet.
