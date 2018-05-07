---
layout: "functions"
page_title: "max function"
sidebar_current: "docs-funcs-numeric-max"
description: |-
  The max function takes one or more numbers and returns the greatest number.
---

# `max` Function

`max` takes one or more numbers and returns the greatest number from the set.

## Examples

```
> max(12, 54, 3)
54
```

If the numbers are in a list or set value, use `...` to expand the collection
to individual arguments:

```
> max([12, 54, 3]...)
54
```

## Related Functions

* [`min`](./min.html), which returns the _smallest_ number from a set.
