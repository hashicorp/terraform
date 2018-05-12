---
layout: "functions"
page_title: "coalesce function"
sidebar_current: "docs-funcs-collection-coalesce-x"
description: |-
  The coalesce function takes any number of string arguments and returns the
  first one that isn't empty.
---

# `coalesce` Function

`coalesce` takes any number of string arguments and returns the first one
that isn't empty.

## Examples

```
> coalesce("a", "b")
a
> coalesce("", "b")
b
```

To perform the `coalesce` operation with a list of strings, use the `...`
symbol to expand the list as arguments:

```
> coalesce(["", "b"]...)
b
```

## Related Functions

* [`coalescelist`](./coalescelist.html) performs a similar operation with
  list arguments rather than string arguments.
