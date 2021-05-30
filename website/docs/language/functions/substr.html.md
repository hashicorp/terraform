---
layout: "language"
page_title: "substr - Functions - Configuration Language"
sidebar_current: "docs-funcs-string-substr"
description: |-
  The substr function extracts a substring from a given string by offset and
  length.
---

# `substr` Function

`substr` extracts a substring from a given string by offset and length.

```hcl
substr(string, offset, length)
```

## Examples

```
> substr("hello world", 1, 4)
ello
```

The offset and length are both counted in _unicode characters_ rather than
bytes:

```
> substr("🤔🤷", 0, 1)
🤔
```

The offset index may be negative, in which case it is relative to the end of
the given string.  The length may be -1, in which case the remainder of the
string after the given offset will be returned.

```
> substr("hello world", -5, -1)
world
```
