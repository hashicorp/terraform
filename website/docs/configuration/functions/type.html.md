---
layout: "functions"
page_title: "type - Functions - Configuration Language"
sidebar_current: "docs-funcs-type-type"
description: |-
  The type function detects the type of a given value and returns it as string.
---

# `type` Function

-> **Note:** This page is about Terraform 0.12 and later, and documents a
feature that did not exist in older versions. For other information about
Terraform 0.11 and earlier, see
[0.11 Configuration Language](../../configuration-0-11/index.html).

`type` takes any value, detects its [type](../expressions.html#types-and-values)
and returns it as a string.

Possible return values are:

- bool
- list
- map
- null
- number
- object
- set
- string
- tuple

There is no automatic conversion happening when checking a type. Be aware of
these similar but not identical complex types:

- [map](../types.html#map-)
- [object](../types.html#object-)
- [list](../types.html#list-)
- [set](../types.html#set-)
- [tuple](../types.html#tuple-)

## Examples

```
> type(true)
"bool"

> type(list(1, 2, 3))
"list"

> type(map("key", "value"))
"map"

> type(null)
"null"

> type(42)
"number"

> type({ key : "value" })
"object"

> type(set(1, 2, 3))
"set"

> type("hello world")
"string"

> type([1, 2, 3])
"tuple"
```
