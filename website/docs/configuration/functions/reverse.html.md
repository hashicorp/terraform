---
layout: "functions"
page_title: "reverse - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-reverse"
description: |-
  The reverse function reverses a sequence.
---

# `reverse` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`reverse` takes a sequence and produces a new sequence of the same length
with all of the same elements as the given sequence but in reverse order.

## Examples

```
> reverse([1, 2, 3])
[
  3,
  2,
  1,
]
```

## Related Functions

* [`strrev`](./strrev.html) reverses a string.
