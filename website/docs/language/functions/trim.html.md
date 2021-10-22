---
layout: "language"
page_title: "trim - Functions - Configuration Language"
sidebar_current: "docs-funcs-string-trim"
description: |-
  The trim function removes the specified set of characters from the start and end of
  a given string.
---

# `trim` Function

`trim` removes the specified set of characters from the start and end of the given
string.

```hcl
trim(string, str_character_set)
```

Every occurrence of a character in the second argument is removed from the start 
and end of the string specified in the first argument. 

## Examples

```
> trim("?!hello?!", "!?")
"hello"

> trim("foobar", "far")
"oob"

> trim("   hello! world.!  ", "! ")
"hello! world."
```

## Related Functions

* [`trimprefix`](./trimprefix.html) removes a word from the start of a string.
* [`trimsuffix`](./trimsuffix.html) removes a word from the end of a string.
* [`trimspace`](./trimspace.html) removes all types of whitespace from
  both the start and the end of a string.
