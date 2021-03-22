---
layout: "language"
page_title: listtomap - Functions - Configuration Language
sidebar_current: docs-funcs-collection-listtomap
description: |-
  The listtomap function converts a list (resp tuple) to a map (resp object)
  were the keys of are the indices (resp positions).
---

# `listtomap` Function

-> **Note:** This function is available in Terraform 0.15.? and later.

Do not expect the order of iteration to be preserved after applying this function.

```hcl
listtomap(list)
```

## Examples

```command
> listtomap(["a", "b"])
{
  "0" = "a"
  "1" = "b"
}
> listtomap(["a", 1])
{
  "0" = "a"
  "1" = 1
}
> listtomap([])
{}
```

The order of the elements may not be preserved:
```
> listtomap([0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10])
{
  "0" = 0
  "1" = 1
  "10" = 10
  "2" = 2
  "3" = 3
  "4" = 4
  "5" = 5
  "6" = 6
  "7" = 7
  "8" = 8
  "9" = 9
}
```

## Related Transformations

* Use a [`for`](../expressions/for.html) expression if you need to offset the indices.
