---
layout: "functions"
page_title: "formatlist - Functions - Configuration Language"
sidebar_current: "docs-funcs-string-formatlist"
description: |-
  The formatlist function produces a list of strings by formatting a number of
  other values according to a specification string.
---

# `formatlist` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`formatlist` produces a list of strings by formatting a number of other
values according to a specification string.

```hcl
formatlist(spec, values...)
```

The specification string uses
[the same syntax as `format`](./format.html#specification-syntax).

The given values can be a mixture of list and non-list arguments. Any given
lists must be the same length, which decides the length of the resulting list.

The list arguments are iterated together in order by index, while the non-list
arguments are used repeatedly for each iteration. The format string is evaluated
once per element of the list arguments.

## Examples

```
> formatlist("Hello, %s!", ["Valentina", "Ander", "Olivia", "Sam"])
[
  "Hello, Valentina!",
  "Hello, Ander!",
  "Hello, Olivia!",
  "Hello, Sam!",
]
> formatlist("%s, %s!", "Salutations", ["Valentina", "Ander", "Olivia", "Sam"])
[
  "Salutations, Valentina!",
  "Salutations, Ander!",
  "Salutations, Olivia!",
  "Salutations, Sam!",
]
```

## Related Functions

* [`format`](./format.html) defines the specification syntax used by this
  function and produces a single string as its result.
