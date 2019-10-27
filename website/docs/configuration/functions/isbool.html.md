---
layout: "functions"
page_title: "isbool - Functions - Configuration Language"
sidebar_current: "docs-funcs-type-isbool"
description: |-
  The isbool function checks if a given value is of type bool and returns
  either true or false.
---

# `isbool` Function

-> **Note:** This page is about Terraform 0.12 and later, and documents a
feature that did not exist in older versions. For other information about
Terraform 0.11 and earlier, see
[0.11 Configuration Language](../../configuration-0-11/index.html).

`isbool` checks if a given value is of [type](../types.html) `bool` and returns
either `true` or `false`.

## Examples

```
> isbool(true)
true

> isbool(false)
true

> isbool(42)
false
```
