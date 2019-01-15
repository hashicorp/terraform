---
layout: "functions"
page_title: "setproduct - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-setproduct"
description: |-
  The setproduct function finds all of the possible combinations of elements
  from all of the given sets by computing the cartesian product.
---

# `setproduct` Function

The `setproduct` function finds all of the possible combinations of elements
from all of the given sets by computing the
[cartesian product](https://en.wikipedia.org/wiki/Cartesian_product).

```hcl
setproduct(sets...)
```

This function is particularly useful for finding the exhaustive set of all
combinations of members of multiple sets, such as per-application-per-environment
resources.

```
> setproduct(["development", "staging", "production"], ["app1", "app2"])
[
  [
    "development",
    "app1",
  ],
  [
    "development",
    "app2",
  ],
  [
    "staging",
    "app1",
  ],
  [
    "staging",
    "app2",
  ],
  [
    "production",
    "app1",
  ],
  [
    "production",
    "app2",
  ],
]
```

You must past at least two arguments to this function.

Although defined primarily for sets, this function can also work with lists.
If all of the given arguments are lists then the result is a list, preserving
the ordering of the given lists. Otherwise the result is a set. In either case,
the result's element type is a list of values corresponding to each given
argument in turn.

## Examples

There is an example of the common usage of this function above. There are some
other situations that are less common when hand-writing but may arise in
reusable module situations.

If any of the arguments is empty then the result is always empty itself,
similar to how multiplying any number by zero gives zero:

```
> setproduct(["development", "staging", "production"], [])
[]
```

Similarly, if all of the arguments have only one element then the result has
only one element, which is the first element of each argument:

```
> setproduct(["a"], ["b"])
[
  [
    "a",
    "b",
  ],
]
```

Each argument must have a consistent type for all of its elements. If not,
Terraform will attempt to convert to the most general type, or produce an
error if such a conversion is impossible. For example, mixing both strings and
numbers results in the numbers being converted to strings so that the result
elements all have a consistent type:

```
> setproduct(["staging", "production"], ["a", 2])
[
  [
    "staging",
    "a",
  ],
  [
    "staging",
    "2",
  ],
  [
    "production",
    "a",
  ],
  [
    "production",
    "2",
  ],
]
```
