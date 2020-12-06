---
layout: "language"
page_title: "sum - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-sum"
description: |-
  The sum function takes a list or set of numbers and returns the sum of those
  numbers.
---

# `sum` Function

-> **Note:** This page is about Terraform 0.13 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`sum` takes a list or set of numbers and returns the sum of those numbers.


## Examples

```
> sum([10, 13, 6, 4.5])
33.5
```