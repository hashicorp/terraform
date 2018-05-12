---
layout: "functions"
page_title: "sort function"
sidebar_current: "docs-funcs-collection-sort"
description: |-
  The sort function takes a list of strings and returns a new list with those
  strings sorted lexicographically.
---

# `sort` Function

`sort` takes a list of strings and returns a new list with those strings
sorted lexicographically.

The sort is in terms of Unicode codepoints, with higher codepoints appearing
after lower ones in the result.

## Examples

```
> sort(["e", "d", "a", "x"])
[
  "a",
  "d",
  "e",
  "x",
]
```
