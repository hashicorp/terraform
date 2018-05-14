---
layout: "functions"
page_title: "transpose function"
sidebar_current: "docs-funcs-collection-transpose"
description: |-
  The transpose function takes a map of lists of strings and swaps the keys
  and values.
---

# `transpose` Function

`transpose` takes a map of lists of strings and swaps the keys and values
to produce a new map of lists of strings.

## Examples

```
> transpose({"a" = ["1", "2"], "b" = ["2", "3"]})
{
  "1" = [
    "a",
  ],
  "2" = [
    "a",
    "b",
  ],
  "3" = [
    "b",
  ],
}
```
