---
layout: "language"
page_title: "setsubtract - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-setsubtract"
description: |-
  The setsubtract function returns a new set containing the elements
  from the first set that are not present in the second set
---

# `setsubtract` Function

The `setsubtract` function returns a new set containing the elements from the first set that are not present in the second set. In other words, it computes the
[relative complement](https://en.wikipedia.org/wiki/Complement_(set_theory)#Relative_complement) of the second set.

```hcl
setsubtract(a, b)
```

## Examples

```
> setsubtract(["a", "b", "c"], ["a", "c"])
[
  "b",
]
```

### Set Difference (Symmetric Difference)

```
> setunion(setsubtract(["a", "b", "c"], ["a", "c", "d"]), setsubtract(["a", "c", "d"], ["a", "b", "c"]))
[
  "b",
  "d",
]
```


## Related Functions

* [`setintersection`](./setintersection.html) computes the _intersection_ of multiple sets
* [`setproduct`](./setproduct.html) computes the _Cartesian product_ of multiple
  sets.
* [`setunion`](./setunion.html) computes the _union_ of
  multiple sets.
