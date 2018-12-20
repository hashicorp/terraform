---
layout: "functions"
page_title: "zipmap - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-zipmap"
description: |-
  The zipmap function constructs a map from a list of keys and a corresponding
  list of values.
---

# `zipmap` Function

`zipmap` constructs a map from a list of keys and a corresponding list of
values.

```hcl
zipmap(keyslist, valueslist)
```

Both `keyslist` and `valueslist` must be of the same length. `keyslist` must
be a list of strings, while `valueslist` can be a list of any type.

Each pair of elements with the same index from the two lists will be used
as the key and value of an element in the resulting map. If the same value
appears multiple times in `keyslist` then the value with the highest index
is used in the resulting map.

## Examples

```
> zipmap(["a", "b"], [1, 2])
{
  "a" = 1,
  "b" = 2,
}
```
