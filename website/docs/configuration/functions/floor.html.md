---
layout: "functions"
page_title: "floor - Functions - Configuration Language"
sidebar_current: "docs-funcs-numeric-floor"
description: |-
  The floor function returns the closest whole number less than or equal to
  the given value.
---

# `floor` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`floor` returns the closest whole number that is less than or equal to the
given value, which may be a fraction.

## Examples

```
> floor(5)
5
> floor(4.9)
4
```

## Related Functions

* [`ceil`](./ceil.html), which rounds to the nearest whole number _greater than_
  or equal.
