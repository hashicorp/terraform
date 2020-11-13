---
layout: "language"
page_title: "trimsuffix - Functions - Configuration Language"
sidebar_current: "docs-funcs-string-trimsuffix"
description: |-
  The trimsuffix function removes the specified suffix from the end of a
  given string.
---

# `trimsuffix` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`trimsuffix` removes the specified suffix from the end of the given string.

## Examples

```
> trimsuffix("helloworld", "world")
hello
```

## Related Functions

* [`trim`](./trim.html) removes characters at the start and end of a string.
* [`trimprefix`](./trimprefix.html) removes a word from the start of a string.
* [`trimspace`](./trimspace.html) removes all types of whitespace from
  both the start and the end of a string.
