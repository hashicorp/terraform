---
layout: "functions"
page_title: "index - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-index"
description: |-
  The index function finds the element index for a given value in a list.
---

# `index` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`index` finds the element index for a given value in a list.

```hcl
index(list, value)
```

The returned index is zero-based. This function produces an error if the given
value is not present in the list.

## Examples

```
> index(["a", "b", "c"], "b")
1
```

## Related Functions

* [`element`](./element.html) retrieves a particular element from a list given
  its index.
