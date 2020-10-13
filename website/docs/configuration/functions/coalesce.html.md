---
layout: "functions"
page_title: "coalesce - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-coalesce-x"
description: |-
  The coalesce function takes any number of arguments and returns the
  first one that isn't null nor empty.
---

# `coalesce` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`coalesce` takes any number of arguments and returns the first one
that isn't null or an empty string.

## Examples

```
> coalesce("a", "b")
a
> coalesce("", "b")
b
> coalesce(1,2)
1
```

To perform the `coalesce` operation with a list of strings, use the `...`
symbol to expand the list as arguments:

```
> coalesce(["", "b"]...)
b
```

## Related Functions

* [`coalescelist`](./coalescelist.html) performs a similar operation with
  list arguments rather than individual arguments.
