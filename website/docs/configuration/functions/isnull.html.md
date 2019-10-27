---
layout: "functions"
page_title: "isnull - Functions - Configuration Language"
sidebar_current: "docs-funcs-type-isnull"
description: |-
  The isnull function checks if a given value is null and returns either true
  or false.
---

# `isnull` Function

-> **Note:** This page is about Terraform 0.12 and later, and documents a
feature that did not exist in older versions. For other information about
Terraform 0.11 and earlier, see
[0.11 Configuration Language](../../configuration-0-11/index.html).

`isnull` checks if a given [value](../expressions.html#types-and-values) is
`null` and returns either `true` or `false`.

## Examples

```
> isnull(null)
true

> isnull(false)
false
```
