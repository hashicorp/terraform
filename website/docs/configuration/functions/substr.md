---
layout: "functions"
page_title: "substr function"
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
