---
layout: "functions"
page_title: "join function"
sidebar_current: "docs-funcs-string-join"
description: |-
  The join function produces a string by concatenating the elements of a list
  with a given delimiter.
---

# `join` Function

`join` produces a string by concatenating together all elements of a given
list of strings with the given delimiter.

```hcl
join(separator, list)
```

## Examples

```
> join(", ", ["foo", "bar", "baz"])
foo, bar, baz
> join(", ", ["foo"])
foo
```

## Related Functions

* [`split`](./split.html) performs the opposite operation: producing a list
  by separating a single string using a given delimiter.
