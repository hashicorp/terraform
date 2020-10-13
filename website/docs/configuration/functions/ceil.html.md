---
layout: "functions"
page_title: "ceil - Functions - Configuration Language"
sidebar_current: "docs-funcs-numeric-ceil"
description: |-
  The ceil function returns the closest whole number greater than or equal to
  the given value.
---

# `ceil` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`ceil` returns the closest whole number that is greater than or equal to the
given value, which may be a fraction.

## Examples

```
> ceil(5)
5
> ceil(5.1)
6
```

## Related Functions

* [`floor`](./floor.html), which rounds to the nearest whole number _less than_
  or equal.
