---
layout: "functions"
page_title: "pow - Functions - Configuration Language"
sidebar_current: "docs-funcs-numeric-pow"
description: |-
  The pow function raises a number to a power.
---

# `pow` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`pow` calculates an exponent, by raising its first argument to the power of the second argument.

## Examples

```
> pow(3, 2)
9
> pow(4, 0)
1
```
