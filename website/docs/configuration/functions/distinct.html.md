---
layout: "functions"
page_title: "distinct - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-distinct"
description: |-
  The distinct function removes duplicate elements from a list.
---

# `distinct` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`distinct` takes a list and returns a new list with any duplicate elements
removed.

The first occurence of each value is retained and the relative ordering of
these elements is preserved.

## Examples

```
> distinct(["a", "b", "a", "c", "d", "b"])
[
  "a",
  "b",
  "c",
  "d",
]
```
