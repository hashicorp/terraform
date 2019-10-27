---
layout: "functions"
page_title: "isobject - Functions - Configuration Language"
sidebar_current: "docs-funcs-type-isobject"
description: |-
  The isobject function checks if a given value is of type object and returns
  either true or false.
---

# `isnumber` Function

-> **Note:** This page is about Terraform 0.12 and later, and documents a
feature that did not exist in older versions. For other information about
Terraform 0.11 and earlier, see
[0.11 Configuration Language](../../configuration-0-11/index.html).

`isobject` checks if a given value is of [type](../types.html) `object` and
returns either `true` or `false`.

There is no automatic conversion happening when checking a type. Be aware of
these similar but not identical complex types:

- [map](../types.html#map-)
- [object](../types.html#object-)

## Examples

```
> isobject({ key : "value" })
true

> isobject(map("key", "value")) # it's a map!
false
```
