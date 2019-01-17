---
layout: "functions"
page_title: "signum - Functions - Configuration Language"
sidebar_current: "docs-funcs-numeric-signum"
description: |-
  The signum function determines the sign of a number.
---

# `signum` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`signum` determines the sign of a number, returning a number between -1 and
1 to represent the sign.

## Examples

```
> signum(-13)
-1
> signum(0)
0
> signum(344)
1
```
