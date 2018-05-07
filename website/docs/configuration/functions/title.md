---
layout: "functions"
page_title: "title function"
sidebar_current: "docs-funcs-string-title"
description: |-
  The title function converts the first letter of each word in a given string
  to uppercase.
---

# `title` Function

`title` converts the first letter of each word in the given string to uppercase.

## Examples

```
> title("hello world")
Hello World
```

This function uses Unicode's definition of letters and of upper- and lowercase.

## Related Functions

* [`upper`](./upper.html) converts _all_ letters in a string to uppercase.
* [`lower`](./lower.html) converts all letters in a string to lowercase.
