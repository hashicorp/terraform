---
layout: "functions"
page_title: "parseint - Functions - Configuration Language"
sidebar_current: "docs-funcs-numeric-parseint"
description: |-
  The parseint function parses the given string as a representation of an integer.
---

# `parseint` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`parseint` parses the given string as a representation of an integer in
the specified base and returns the resulting number. The base must be between 2
and 62 inclusive.

All bases use the arabic numerals 0 through 9 first. Bases between 11 and 36
inclusive use case-insensitive latin letters to represent higher unit values.
Bases 37 and higher use lowercase latin letters and then uppercase latin
letters.

If the given string contains any non-digit characters or digit characters that
are too large for the given base then `parseint` will produce an error.

## Examples

```
> parseint("100", 10)
100

> parseint("FF", 16)
255

> parseint("-10", 16)
-16

> parseint("1011111011101111", 2)
48879

> parseint("aA", 62)
656

> parseint("12", 2)

Error: Invalid function argument

Invalid value for "number" parameter: cannot parse "12" as a base 2 integer.
```

## Related Functions

* [`format`](./format.html) can format numbers and other values into strings,
  with optional zero padding, alignment, etc.
