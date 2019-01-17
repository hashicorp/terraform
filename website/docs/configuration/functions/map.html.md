---
layout: "functions"
page_title: "map - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-map"
description: |-
  The map function constructs a map from some given elements.
---

# `map` Function

~> **This function is deprecated.** From Terraform v0.12, the Terraform
language has built-in syntax for creating maps using the `{` and `}`
delimiters. Use the built-in syntax instead. The `map` function will be
removed in a future version of Terraform.

`map` takes an even number of arguments and returns a map whose elements
are constructed from consecutive pairs of arguments.

## Examples

```
> map("a", "b", "c", "d")
{
  "a" = "b"
  "c" = "d"
]
```

Do not use the above form in Terraform v0.12 or above. Instead, use the
built-in map construction syntax, which achieves the same result:

```
> {"a" = "b", "c" = "d"}
{
  "a" = "b"
  "c" = "d"
]
```

## Related Functions

* [`tomap`](./tomap.html) performs a type conversion to a map type.
