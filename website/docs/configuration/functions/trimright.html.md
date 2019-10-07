---
layout: "functions"
page_title: "trimright - Functions - Configuration Language"
sidebar_current: "docs-funcs-string-trimright"
description: |-
  The trimright function removes the specified characters from the end of a
  given string.
---

# `trimright` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`trimright` removes the specified characters from the end of the given string.

## Examples

```
> trimright("!?hello?!", "!?")
!?hello
```

## Related Functions

* [`trim`](./trim.html) removes characters at the start and end of a string.
* [`trimleft`](./trimleft.html) removes characters at the start of a string.
* [`trimspace`](./trimspace.html) removes all types of whitespace from
  both the start and the end of a string.
