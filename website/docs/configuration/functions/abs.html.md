---
layout: "functions"
page_title: "abs - Functions - Configuration Language"
sidebar_current: "docs-funcs-numeric-abs"
description: |-
  The abs function returns the absolute value of the given number.
---

# `abs` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`abs` returns the absolute value of the given number. In other words, if the
number is zero or positive then it is returned as-is, but if it is negative
then it is multiplied by -1 to make it positive before returning it.

## Examples

```
> abs(23)
23
> abs(0)
0
> abs(-12.4)
12.4
```
