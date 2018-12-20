---
layout: "functions"
page_title: "compact - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-compact"
description: |-
  The compact function removes empty string elements from a list.
---

# `compact` Function

`compact` takes a list of strings and returns a new list with any empty string
elements removed.

## Examples

```
> compact(["a", "", "b", "c"])
[
  "a",
  "b",
  "c",
]
```
