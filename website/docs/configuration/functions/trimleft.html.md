---
layout: "functions"
page_title: "trimleft - Functions - Configuration Language"
sidebar_current: "docs-funcs-string-trimleft"
description: |-
  The trimleft function removes the specified characters from the start of a
  given string.
---

# `trimleft` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`trimleft` removes the specified characters from the start of the given string.

## Examples

```
> trimleft("!?hello?!", "!?")
hello?!
```

## Related Functions

* [`trim`](./trim.html) removes characters at the start and end of a string.
* [`trimright`](./trimright.html) removes characters at the end of a string.
* [`trimspace`](./trimspace.html) removes all types of whitespace from
  both the start and the end of a string.
