---
layout: "functions"
page_title: "isnumber - Functions - Configuration Language"
sidebar_current: "docs-funcs-type-isnumber"
description: |-
  The isnumber function checks if a given value is of type number and returns
  either true or false.
---

# `isnumber` Function

-> **Note:** This page is about Terraform 0.12 and later, and documents a
feature that did not exist in older versions. For other information about
Terraform 0.11 and earlier, see
[0.11 Configuration Language](../../configuration-0-11/index.html).

`isnumber` checks if a given value is of [type](../types.html) `number` and
returns either `true` or `false`.

## Examples

```
> isnumber(42)
true

> isnumber("hello world")
false
```
