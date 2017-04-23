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
  byte_match_tuple {
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
* `byte_match_tuple` - (Optional)Settings for the ByteMatchSet, such as the bytes (typically a string that corresponds with ASCII characters) that you want AWS WAF to search for in web requests. ByteMatchTuple documented below.

ByteMatchTuple(byte_match_tuple) support the following:

* `field_to_match` - (Required) Settings for the ByteMatchTuple. FieldToMatch documented below.
* `positional_constraint` - (Required) Within the portion of a web request that you want to search.
* `target_string` - (Required) The value that you want AWS WAF to search for. The maximum length of the value is 50 bytes.
* `text_transformation` - (Required) The formatting way for web request.

FieldToMatch(field_to_match) support following:

* `data` - (Optional) When the value of Type is HEADER, enter the name of the header that you want AWS WAF to search, for example, User-Agent or Referer. If the value of Type is any other value, omit Data.
* `type` - (Required) The part of the web request that you want AWS WAF to search for a specified string.

## Remarks

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the WAF ByteMatchSet.
