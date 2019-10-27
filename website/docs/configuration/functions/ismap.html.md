---
layout: "functions"
page_title: "ismap - Functions - Configuration Language"
sidebar_current: "docs-funcs-type-ismap"
description: |-
  The ismap function checks if a given value is of type map and returns either
  true or false.
---

# `ismap` Function

-> **Note:** This page is about Terraform 0.12 and later, and documents a
feature that did not exist in older versions. For other information about
Terraform 0.11 and earlier, see
[0.11 Configuration Language](../../configuration-0-11/index.html).

`ismap` checks if a given value is of [type](../types.html) `map` and returns
either `true` or `false`.

There is no automatic conversion happening when checking a type. Be aware of
these similar but not identical complex types:

- [map](../types.html#map-)
- [object](../types.html#object-)

## Examples

```
> ismap(map("key", "value"))
true

> ismap({ key : "value" }) # it's an object!
false
```
