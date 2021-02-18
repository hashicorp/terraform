---
layout: "language"
page_title: "trimsuffix - Functions - Configuration Language"
sidebar_current: "docs-funcs-string-trimsuffix"
description: |-
  The trimsuffix function removes the specified suffix from the end of a
  given string.
---

# `trimsuffix` Function

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
