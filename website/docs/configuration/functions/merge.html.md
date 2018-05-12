---
layout: "functions"
page_title: "merge function"
sidebar_current: "docs-funcs-collection-merge"
description: |-
  The merge function takes an arbitrary number of maps and returns a single
  map after merging the keys from each argument.
---

# `merge` Function

`merge` takes an arbitrary number of maps and returns a single map that
contains a merged set of elements from all of the maps.

If more than one given map defines the same key then the one that is later
in the argument sequence takes precedence.

## Examples

```
> merge({"a"="b", "c"="d"}, {"e"="f", "c"="z"})
{
  "a" = "b"
  "c" = "z"
  "e" = "f"
}
```
