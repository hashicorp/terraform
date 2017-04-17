---
layout: "aws"
page_title: "AWS: wafregional_byte_match_set"
sidebar_current: "docs-aws-resource-wafregional-bytematchset"
description: |-
  Provides a AWS WAF Regional ByteMatchSet resource for use with ALB.
---

# aws\_wafregional\_byte\_match\_set

Provides a WAF Regional Byte Match Set Resource for use with Application Load Balancer.

## Example Usage

```
resource "aws_wafregional_byte_match_set" "byte_set" {
  name = "tf_waf_byte_match_set"
  byte_match_tuples {
    text_transformation = "NONE"
    target_string = "badrefer1"
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

* `name` - (Required) The name or description of the ByteMatchSet.
* `byte_match_tuples` - Settings for the ByteMatchSet, such as the bytes (typically a string that corresponds with ASCII characters) that you want AWS WAF to search for in web requests. 

## Remarks

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the WAF ByteMatchSet.
