---
layout: "functions"
page_title: "distinct function"
sidebar_current: "docs-funcs-collection-distinct"
description: |-
  The distinct function removes duplicate elements from a list.
---

# `distinct` Function

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
