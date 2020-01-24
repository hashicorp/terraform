---
layout: "functions"
page_title: "flipmap - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-flipmap"
description: |-
  The flipmap function flips keys and values of a map or object of strings.
---

# `flipmap` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`flipmap` flips keys and values of a map or object of strings.

```hcl
flipmap(inputObject)
```

`inputObject` must be `object` or `map` wich contains only `string` values.

## Examples

```
> flipmap({ hello = "world" })
{
  world = "hello"
}
```
