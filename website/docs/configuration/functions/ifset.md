---
layout: "functions"
page_title: "ifset - Functions - Configuration Language"
sidebar_current: "docs-funcs-numeric-ifset"
description: |-
  The ifset function returns the specified value or the default if the key is not defined.
---

# `ifset` Function

-> **Note:** This page is about Terraform 0.12 and later

`ifset` takes a map, a key as string and a default value and returns the value of the key in the map or the default value if the key does not exist.

## Examples

```
> ifset({ a="1", b="2"},"a","3")
1
> ifset({ a="1", b="2"},"d","3")
3
```

```
> ifset({ a= { x = "1" },b= { y = "2" }},"a",{ z = "3"} )
{
  "x" = "1"
}
> ifset({ a= { x = "1" },b= { y = "2" }},"c",{ z = "3"} )
{
  "z" = "1"
}
```
