---
layout: "functions"
page_title: "split - Functions - Configuration Language"
sidebar_current: "docs-funcs-string-split"
description: |-
  The split function produces a list by dividing a given string at all
  occurences of a given separator.
---

# `split` Function

`split` produces a list by dividing a given string at all occurences of a
given separator.

```hcl
split(separator, string)
```

## Examples

```
> split(",", "foo,bar,baz")
[
  "foo",
  "bar",
  "baz",
]
> split(",", "foo")
[
  "foo",
]
> split(",", "")
[
  "",
]
```

## Related Functions

* [`join`](./join.html) performs the opposite operation: producing a string
  joining together a list of strings with a given separator.
