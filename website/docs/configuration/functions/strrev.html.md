---
layout: "functions"
page_title: "strrev - Functions - Configuration Language"
sidebar_current: "docs-funcs-string-strrev"
description: |-
  The strrev function reverses a string.
---

# `strrev` Function

`strrev` reverses the characters in a string.
Note that the characters are treated as _Unicode characters_ (in technical terms, Unicode [grapheme cluster boundaries](https://unicode.org/reports/tr29/#Grapheme_Cluster_Boundaries) are respected).

```hcl
strrev(string)
```

## Examples

```
> strrev("hello")
olleh
> strrev("a ☃")
☃ a
```

## Related Functions

* [`reverse`](./reverse.html) reverses a sequence.
