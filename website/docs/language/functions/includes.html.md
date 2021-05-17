---
layout: "language"
page_title: "includes - Functions - Configuration Language"
sidebar_current: "docs-funcs-string-includes"
description: |-
  The includes function determines whether a given string may be found within another string.
---

# `includes` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`includes` returns whether a substring is within another string.

```hcl
includes(string, substr)
```

## Examples

```
> includes("hello world", "wor")
true
```

```
> includes("hello world", "wod")
false
```
