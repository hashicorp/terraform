---
layout: functions
page_title: all - Functions - Configuration Language
sidebar_current: docs-funcs-collection-all
description: |-
  The all function determines whether all elements of a collection are true or
  "true". If the collection is empty, it returns true.
---

# `all` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`all` returns `true` if all elements in a given collection are `true` or
`"true"`. It also returns `true` if the collection is empty.

```hcl
all(list)
```

## Examples

```command
> all(["true", true])
true
> all([true, false])
false
```
