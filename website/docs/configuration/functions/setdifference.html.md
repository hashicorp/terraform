---
layout: "functions"
page_title: "setdifference - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-setsubtract"
description: |-
  The setdifference function returns a new set containing elements
  that appear in any of the given sets but not multiple
---

# `setdifference` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

The `setdifference` function returns a new set containing elements that appear in any of the given sets but not multiple. In other words, it computes the
[symmetric difference](https://en.wikipedia.org/wiki/Symmetric_difference) of the sets.

```hcl
setdifference(a, b, c)
```

## Examples

```
> setdifference(["a", "b"], ["a", "c"], ["a", "d"])
[
  "b",
  "c",
  "d",
]
```

## Related Functions

* [`setintersection`](./setintersection.html) computes the _intersection_ of multiple sets
* [`setproduct`](./setproduct.html) computes the _Cartesian product_ of multiple
  sets.
* [`setsubtract`](./setdifference.html) computes the _relative complement_ of two sets
* [`setunion`](./setunion.html) computes the _union_ of
  multiple sets.
