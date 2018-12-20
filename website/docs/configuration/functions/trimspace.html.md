---
layout: "functions"
page_title: "trimspace - Functions - Configuration Language"
sidebar_current: "docs-funcs-string-trimspace"
description: |-
  The trimspace function removes space characters from the start and end of
  a given string.
---

# `trimspace` Function

`trimspace` removes any space characters from the start and end of the given
string.

This function follows the Unicode definition of "space", which includes
regular spaces, tabs, newline characters, and various other space-like
characters.

## Examples

```
> trimspace("  hello\n\n")
hello
```

## Related Functions

* [`chomp`](./chomp.html) removes just line ending characters from the _end_ of
  a string.
