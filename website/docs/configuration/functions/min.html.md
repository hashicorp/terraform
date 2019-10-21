---
layout: "functions"
page_title: "min - Functions - Configuration Language"
sidebar_current: "docs-funcs-numeric-min"
description: |-
  The min function takes one or more numbers and returns the smallest number.
---

# `min` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`min` takes one or more numbers and returns the smallest number from the set.

## Examples

```
> min(12, 54, 3)
3
```

If the numbers are in a list or set value, use `...` to expand the collection
to individual arguments:

```
> min([12, 54, 3]...)
3
```

## Related Functions

* [`max`](./max.html), which returns the _greatest_ number from a set.
