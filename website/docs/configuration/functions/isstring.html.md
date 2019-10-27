---
layout: "functions"
page_title: "isstring - Functions - Configuration Language"
sidebar_current: "docs-funcs-type-isstring"
description: |-
  The isstring function checks if a given value is of type string and returns
  either true or false.
---

# `isstring` Function

-> **Note:** This page is about Terraform 0.12 and later, and documents a
feature that did not exist in older versions. For other information about
Terraform 0.11 and earlier, see
[0.11 Configuration Language](../../configuration-0-11/index.html).

`isstring` checks if a given value is of [type](../types.html) `string` and
returns either `true` or `false`.

## Examples

```
> isstring("hello world")
true

> isstring(42)
false
```
