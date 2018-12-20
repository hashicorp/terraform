---
layout: "functions"
page_title: "min - Functions - Configuration Language"
sidebar_current: "docs-funcs-numeric-min"
description: |-
  The min function takes one or more numbers and returns the smallest number.
---

# `min` Function

`min` takes one or more numbers and returns the smallest number from the set.

## Examples

```
> min(12, 54, 3)
3
```

If the numbers are in a list or set value, use `...` to expand the collection
to individual arguments:

```
> min([12, 54, 3]...)
3
```

## Related Functions

* [`max`](./max.html), which returns the _greatest_ number from a set.
