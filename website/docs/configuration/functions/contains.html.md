---
layout: "functions"
page_title: "contains - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-contains"
description: |-
  The contains function determines whether a list contains a given value.
---

# `contains` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`contains` determines whether a given list contains a given single value
as one of its elements.

```hcl
contains(list, value)
```

## Examples

```
> contains(["a", "b", "c"], "a")
true
> contains(["a", "b", "c"], "d")
false
```
