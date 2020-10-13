---
layout: "functions"
page_title: "concat - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-concat"
description: |-
  The concat function combines two or more lists into a single list.
---

# `concat` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`concat` takes two or more lists and combines them into a single list.

## Examples

```
> concat(["a", ""], ["b", "c"])
[
  "a",
  "",
  "b",
  "c",
]
```
