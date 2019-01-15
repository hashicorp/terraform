---
layout: "functions"
page_title: "sethaselement - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-sethaselement"
description: |-
  The sethaselement function tests whether a given value is in a given set.
---

# `sethaselement` Function

The `sethaselement` function tests whether a given value is in a given set.

```hcl
sethaselement(set, value)
```

## Examples

```
> sethaselement(["a", "b"], "b")
true
> sethaselement(["a", "b"], "c")
false
```

## Related Functions

* [`setintersection`](./setintersection.html) computes the _intersection_ of
  multiple sets.
* [`setproduct`](./setproduct.html) computes the _cartesian product_ of multiple
  sets.
* [`setunion`](./setunion.html) computes the _union_ of
  multiple sets.
