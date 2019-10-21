---
layout: "functions"
page_title: "setunion - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-setunion"
description: |-
  The setunion function takes multiple sets and produces a single set
  containing the elements from all of the given sets.
---

# `setunion` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

The `setunion` function takes multiple sets and produces a single set
containing the elements from all of the given sets. In other words, it
computes the [union](https://en.wikipedia.org/wiki/Union_(set_theory)) of
the sets.

```hcl
setunion(sets...)
```

## Examples

```
> setunion(["a", "b"], ["b", "c"], ["d"])
[
  "d",
  "b",
  "c",
  "a",
]
```

The given arguments are converted to sets, so the result is also a set and
the ordering of the given elements is not preserved.

## Related Functions

* [`contains`](./contains.html) tests whether a given list or set contains
  a given element value.
* [`setintersection`](./setintersection.html) computes the _intersection_ of
  multiple sets.
* [`setproduct`](./setproduct.html) computes the _Cartesian product_ of multiple
  sets.
