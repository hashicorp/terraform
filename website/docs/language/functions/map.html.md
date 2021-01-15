---
layout: "language"
page_title: "map - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-map"
description: |-
  The map function constructs a map from some given elements.
---

# `map` Function

The `map` function is no longer available. Prior to Terraform v0.12 it was
the only available syntax for writing a literal map inside an expression,
but Terraform v0.12 introduced a new first-class syntax.

To update an expression like `map("a", "b", "c", "d")`, write the following instead:

```
tomap({
  a = "b"
  c = "d"
})
```

The `{ ... }` braces construct an object value, and then the `tomap` function
then converts it to a map. For more information on the value types in the
Terraform language, see [Type Constraints](/docs/language/expressions/types.html).

## Related Functions

* [`tomap`](./tomap.html) converts an object value to a map.
* [`zipmap`](./zipmap.html) constructs a map dynamically, by taking keys from
  one list and values from another list.
