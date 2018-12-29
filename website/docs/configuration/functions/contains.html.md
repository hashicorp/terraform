---
layout: "functions"
page_title: "contains - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-contains"
description: |-
  The contains function determines whether a list contains a given value.
---

# `contains` Function

`contains` determines whether a given list contains a given single value
as one of its elements.

```hcl
contains(list, value)
```

## Examples

```
> contains(["a", "b", "c"], "a")
true
> contains(["a", "b", "c"], "d")
false
```
