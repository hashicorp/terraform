---
layout: "functions"
page_title: "indent - Functions - Configuration Language"
sidebar_current: "docs-funcs-string-indent"
description: |-
  The indent function adds a number of spaces to the beginnings of all but the
  first line of a given multi-line string.
---

# `indent` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`indent` adds a given number of spaces to the beginnings of all but the first
line in a given multi-line string.

```hcl
indent(num_spaces, string)
```

## Examples

This function is useful for inserting a multi-line string into an
already-indented context in another string:

```
> "  items: %{indent(2, "[\n  foo,\n  bar,\n]\n")}"
  items: [
    foo,
    bar,
  ]
```

The first line of the string is not indented so that, as above, it can be
placed after an introduction sequence that has already begun the line.
