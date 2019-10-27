---
layout: "functions"
page_title: "islist - Functions - Configuration Language"
sidebar_current: "docs-funcs-type-islist"
description: |-
  The islist function checks if a given value is of type list and returns
  either true or false.
---

# `islist` Function

-> **Note:** This page is about Terraform 0.12 and later, and documents a
feature that did not exist in older versions. For other information about
Terraform 0.11 and earlier, see
[0.11 Configuration Language](../../configuration-0-11/index.html).

`islist` checks if a given value is of [type](../types.html) `list` and returns
either `true` or `false`.

There is no automatic conversion happening when checking a type. Be aware of
these similar but not identical complex types:

- [list](../types.html#list-)
- [set](../types.html#set-)
- [tuple](../types.html#tuple-)

## Examples

```
> islist(list(1, 2, 3))
true

> islist([1, 2, 3]) # it's a tuple!
false

> islist(set(1, 2, 3) # it's a set!
false
```
