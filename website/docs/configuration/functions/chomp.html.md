---
layout: "functions"
page_title: "chomp - Functions - Configuration Language"
sidebar_current: "docs-funcs-string-chomp"
description: |-
  The chomp function removes newline characters at the end of a string.
---

# `chomp` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`chomp` removes newline characters at the end of a string.

This can be useful if, for example, the string was read from a file that has
a newline character at the end.

## Examples

```
> chomp("hello\n")
hello
> chomp("hello\r\n")
hello
> chomp("hello\n\n")
hello
```

## Related Functions

* [`trimspace`](./trimspace.html), which removes all types of whitespace from
  both the start and the end of a string.
