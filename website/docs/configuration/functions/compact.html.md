---
layout: "functions"
page_title: "compact - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-compact"
description: |-
  The compact function removes empty string elements from a list.
---

# `compact` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`compact` takes a list of strings and returns a new list with any empty string
elements removed.

## Examples

```
> compact(["a", "", "b", "c"])
[
  "a",
  "b",
  "c",
]
```
