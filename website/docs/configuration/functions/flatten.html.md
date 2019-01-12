---
layout: "functions"
page_title: "flatten - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-flatten"
description: |-
  The flatten function eliminates nested lists from a list.
---

# `flatten` Function

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
