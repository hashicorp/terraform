---
layout: "functions"
page_title: "flatten - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-flatten"
description: |-
  The flatten function eliminates nested lists from a list.
---

# `flatten` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`flatten` takes a list and replaces any elements that are lists with a
flattened sequence of the list contents.

## Examples

```
> flatten([["a", "b"], [], ["c"]])
["a", "b", "c"]
```

If any of the nested lists also contain directly-nested lists, these too are
flattened recursively:

```
> flatten([[["a", "b"], []], ["c"]])
["a", "b", "c"]
```

Indirectly-nested lists, such as those in maps, are _not_ flattened.
