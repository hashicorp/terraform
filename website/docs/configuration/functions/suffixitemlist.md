---
layout: "functions"
page_title: "suffixitemlist - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-suffixitemlist"
description: |-
  The suffixitemlist function helps to suffix each items in a list.
---

# `suffixitemlist` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`suffixitemlist` produces a new list by adding the suffix to each items of a given
list.

```hcl
suffixitemlist(list, suffix)
```

## Examples

```
> suffixitemlist(["b", "c"], ".")
[
  "a.",
  "b.",
]
```

## Related Functions

* [`prefixitemlist`](./suffixitemlist.html) performs the same operation but with a prefix.
