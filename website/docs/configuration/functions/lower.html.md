---
layout: "functions"
page_title: "lower - Functions - Configuration Language"
sidebar_current: "docs-funcs-string-lower"
description: |-
  The lower function converts all cased letters in the given string to lowercase.
---

# `lower` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`lower` converts all cased letters in the given string to lowercase.

## Examples

```
> lower("HELLO")
hello
> lower("АЛЛО!")
алло!
```

This function uses Unicode's definition of letters and of upper- and lowercase.

## Related Functions

* [`upper`](./upper.html) converts letters in a string to _uppercase_.
* [`title`](./title.html) converts the first letter of each word in a string to uppercase.
