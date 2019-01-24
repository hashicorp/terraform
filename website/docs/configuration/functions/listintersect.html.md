---
layout: "functions"
page_title: "listintersect - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-list"
description: |-
  The listintersect function gets intersectionof two lists.
---

# `listintersect` Function

`listintersect` takes two list as argument and return a list with elements common in both provided lists.

## Examples

```
> listintersect(list("a", "b", "c"), list("b", "d", "e"))
[
  b
]
> listintersect(list("a", "b", "c"), list("b", "c", "e"))
[
  b,
  c
]
```
