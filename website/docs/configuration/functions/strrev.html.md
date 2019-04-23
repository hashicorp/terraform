---
layout: "functions"
page_title: "strrev - Functions - Configuration Language"
sidebar_current: "docs-funcs-string-strrev"
description: |-
  The strrev function reverses a string.
---

# `strrev` Function

`strrev` reverses a string.
Unicode [grapheme cluster boundaries](https://unicode.org/reports/tr29/#Grapheme_Cluster_Boundaries) are respected.

## Examples

```
> strrev("hello")
olleh
> strrev("a ☃")
☃ a
```
