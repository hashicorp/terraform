---
layout: "functions"
page_title: "prefixitemlist - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-prefixitemlist"
description: |-
  The prefixitemlist function helps to prefix each items in a list.
---

# `prefixitemlist` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`prefixitemlist` produces a new list by adding the prefix to each items of a given
list.

```hcl
prefixitemlist(list, prefix)
```

## Examples

```
> prefixitemlist(["b", "c"], ".")
[
  ".a",
  ".b",
]
```

## Related Functions

* [`suffixitemlist`](./suffixitemlist.html) performs the same operation but with a suffix.
