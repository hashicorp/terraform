---
layout: "functions"
page_title: "lookup - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-lookup"
description: |-
  The lookup function retrieves an element value from a map given its key.
---

# `lookup` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`lookup` retrieves the value of a single element from a map, given its key.
If the given key does not exist, a the given default value is returned instead.

```
lookup(map, key, default)
```

-> For historical reasons, the `default` parameter is actually optional. However,
omitting `default` is deprecated since v0.7 because that would then be
equivalent to the native index syntax, `map[key]`.

## Examples

```
> lookup({a="ay", b="bee"}, "a", "what?")
ay
> lookup({a="ay", b="bee"}, "c", "what?")
what?
```

## Related Functions

* [`element`](./element.html) retrieves a value from a _list_ given its _index_.
