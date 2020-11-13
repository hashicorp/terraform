---
layout: "language"
page_title: alltrue - Functions - Configuration Language
sidebar_current: docs-funcs-collection-alltrue
description: |-
  The alltrue function determines whether all elements of a collection
  are true or "true". If the collection is empty, it returns true.
---

# `alltrue` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`alltrue` returns `true` if all elements in a given collection are `true`
or `"true"`. It also returns `true` if the collection is empty.

```hcl
alltrue(list)
```

## Examples

```command
> alltrue(["true", true])
true
> alltrue([true, false])
false
```
