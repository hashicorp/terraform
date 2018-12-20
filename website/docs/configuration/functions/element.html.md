---
layout: "functions"
page_title: "element - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-element"
description: |-
  The element function retrieves a single element from a list.
---

# `element` Function

`element` retrieves a single element from a list.

```hcl
element(list, index)
```

The index is zero-based. This function produces an error if used with an
empty list.

Use the built-in index syntax `list[index]` in most cases. Use this function
only for the special additional "wrap-around" behavior described below.

## Examples

```
> element(["a", "b", "c"], 1)
b
```

If the given index is greater than the length of the list then the index is
"wrapped around" by taking the index modulo the length of the list:

```
> element(["a", "b", "c"], 3)
a
```

## Related Functions

* [`index`](./index.html) finds the index for a particular element value.
* [`lookup`](./lookup.html) retrieves a value from a _map_ given its _key_.
