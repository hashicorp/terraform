---
layout: "functions"
page_title: "chunklist - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-chunklist"
description: |-
  The chunklist function splits a single list into fixed-size chunks, returning
  a list of lists.
---

# `chunklist` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`chunklist` splits a single list into fixed-size chunks, returning a list
of lists.

```hcl
chunklist(list, chunk_size)
```

## Examples

```
> chunklist(["a", "b", "c", "d", "e"], 2)
[
  [
    "a",
    "b",
  ],
  [
    "c",
    "d",
  ],
  [
    "e",
  ],
]
> chunklist(["a", "b", "c", "d", "e"], 1)
[
  [
    "a",
  ],
  [
    "b",
  ],
  [
    "c",
  ],
  [
    "d",
  ],
  [
    "e",
  ],
]
```
