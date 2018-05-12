---
layout: "functions"
page_title: "slice function"
sidebar_current: "docs-funcs-collection-slice"
description: |-
  The slice function extracts some consecutive elements from within a list.
---

# `slice` Function

`slice` extracts some consecutive elements from within a list.

```hcl
slice(list, startindex, endindex)
```

`startindex` is inclusive, while `endindex` is exclusive. This function returns
an error if either index is outside the bounds of valid indices for the given
list.

## Examples

```
> slice(["a", "b", "c", "d"], 1, 3)
[
  "b",
  "c",
]
```

## Related Functions

* [`substr`](./substr.html) performs a similar function for characters in a
  string, although it uses a length instead of an end index.
