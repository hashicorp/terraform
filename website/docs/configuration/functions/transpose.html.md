---
layout: "functions"
page_title: "transpose - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-transpose"
description: |-
  The transpose function takes a map of lists of strings and swaps the keys
  and values.
---

# `transpose` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`transpose` takes a map of lists of strings and swaps the keys and values
to produce a new map of lists of strings.

## Examples

```
> transpose({"a" = ["1", "2"], "b" = ["2", "3"]})
{
  "1" = [
    "a",
  ],
  "2" = [
    "a",
    "b",
  ],
  "3" = [
    "b",
  ],
}
```
