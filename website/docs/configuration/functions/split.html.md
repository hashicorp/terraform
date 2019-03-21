---
layout: "functions"
page_title: "split - Functions - Configuration Language"
sidebar_current: "docs-funcs-string-split"
description: |-
  The split function produces a list by dividing a given string at all
  occurrences of a given separator.
---

# `split` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`split` produces a list by dividing a given string at all occurrences of a
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
