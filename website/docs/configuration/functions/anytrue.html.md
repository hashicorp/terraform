---
layout: "language"
page_title: anytrue - Functions - Configuration Language
sidebar_current: docs-funcs-collection-anytrue
description: |-
  The anytrue function determines whether any element of a collection
  is true or "true". If the collection is empty, it returns false.
---

# `anytrue` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`anytrue` returns `true` if any element in a given collection is `true`
or `"true"`. It also returns `false` if the collection is empty.

```hcl
anytrue(list)
```

## Examples

```command
> anytrue(["true"])
true
> anytrue([true])
true
> anytrue([true, false])
true
> anytrue([])
false
```
