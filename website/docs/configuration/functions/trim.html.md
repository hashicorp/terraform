---
layout: "language"
page_title: "trim - Functions - Configuration Language"
sidebar_current: "docs-funcs-string-trim"
description: |-
  The trim function removes the specified characters from the start and end of
  a given string.
---

# `trim` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`trim` removes the specified characters from the start and end of the given
string.

## Examples

```
> trim("?!hello?!", "!?")
hello
```

## Related Functions

* [`trimprefix`](./trimprefix.html) removes a word from the start of a string.
* [`trimsuffix`](./trimsuffix.html) removes a word from the end of a string.
* [`trimspace`](./trimspace.html) removes all types of whitespace from
  both the start and the end of a string.
