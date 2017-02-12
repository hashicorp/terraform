---
layout: "aws"
page_title: "AWS: waf_byte_match_set"
sidebar_current: "docs-aws-resource-waf-bytematchset"
description: |-
  Provides a AWS WAF ByteMatchSet resource.
---

# aws\_waf\_byte\_match\_set

Provides a WAF Byte Match Set Resource

## Example Usage

```
resource "aws_waf_byte_match_set" "byte_set" {
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
