---
layout: "functions"
page_title: "coalescelist - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-coalescelist"
description: |-
  The coalescelist function takes any number of list arguments and returns the
  first one that isn't empty.
---

# `coalescelist` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`coalescelist` takes any number of list arguments and returns the first one
that isn't empty.

## Examples

```
> coalescelist(["a", "b"], ["c", "d"])
[
  "a",
  "b",
]
> coalescelist([], ["c", "d"])
[
  "c",
  "d",
]
```

To perform the `coalescelist` operation with a list of lists, use the `...`
symbol to expand the outer list as arguments:

```
> coalescelist([[], ["c", "d"]]...)
[
  "c",
  "d",
]
```

## Related Functions

* [`coalesce`](./coalesce.html) performs a similar operation with string
  arguments rather than list arguments.
