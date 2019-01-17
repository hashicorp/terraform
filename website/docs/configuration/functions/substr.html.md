---
layout: "functions"
page_title: "substr - Functions - Configuration Language"
sidebar_current: "docs-funcs-string-substr"
description: |-
  The substr function extracts a substring from a given string by offset and
  length.
---

# `substr` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

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
