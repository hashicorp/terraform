---
layout: "functions"
page_title: "setsymmetricdifference - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-setsymmetricdifference"
description: |-
  The setsymmetricdifference function takes two sets and produces a single
  set containing the elements that are unique to only one set.
---

# `setsymmetricdifference` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).


The `setsymmetricdifference` function takes two sets and produces a single
set containing the elements that are unique to only one set.
In other words, it computes the
[symmetric difference](https://en.wikipedia.org/wiki/Symmetric_difference) of
the sets.

```hcl
setsymmetricdifference(set1, set2)
```

## Examples

```
> setsymmetricdifference(["a", "b"], ["b", "c"])
[
  "a",
  "c",
]
```

The given arguments are converted to sets and the result is also a set, so
`setsymmetricdifference` does not preserve the ordering of elements.

## Related Functions

* [`setintersection`](./setintersection.html) finds all common elements in sets and returns them as a set
* [`contains`](./contains.html) tests whether a given list or set contains
  a given element value.
* [`setproduct`](./setproduct.html) computes the _cartesian product_ of multiple
  sets.
* [`setunion`](./setunion.html) computes the _union_ of
  multiple sets.
