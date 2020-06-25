---
layout: "functions"
page_title: "deepmerge - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-merge"
description: |-
  The deepmerge function takes an arbitrary number maps or objects, and returns a
  single map or object that contains a merged set of elements from all
  arguments.

  The behavior is exactly the same as `merge`, but it will recurse in to objects, and supports
  partial updates of nested objects.
---

# `deepmerge` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`deepmerge` takes an arbitrary number of maps or objects, and returns a single map
or object that contains a merged set of elements from all arguments.

The behavior is exactly the same as `merge`, but it will recurse in to objects, and supports
partial updates of nested objects.

## Null Values
Any properties that are set to a `null` value will be removed from the final object/map. If you wish
to keep these properties, make sure to fill them with a default value (ex: `{}`)

## Examples

```
> deepmerge({a={a="a",a2="a2"},b="b",c={c={c="c"}}}, {a={a="a-updated",a3="new-prop"}, b="b",c={c={c1="new-prop"}}})
{
  "a" = {
    "a" = "a-updated"
    "a2" = "a2"
    "a3" = "new-prop"
  }
  "b" = "b"
}
```

```
> deepmerge({a={a="a",a2="a2"},b="b",c={c={c="c"}}}, {a={a="a-updated",a3="new-prop"}, b="b",c={c={c1="new-prop"}}})
{
  "a" = {
    "a" = "a-updated"
    "a2" = "a2"
    "a3" = "new-prop"
  }
  "b" = "b"
  "c" = {
    "c" = {
      "c" = "c"
      "c1" = "new-prop"
    }
  }
}
```
