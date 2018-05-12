---
layout: "functions"
page_title: "replace function"
sidebar_current: "docs-funcs-string-replace"
description: |-
  The replace function searches a given string for another given substring,
  and replaces all occurences with a given replacement string.
---

# `replace` Function

`replace` searches a given string for another given substring, and replaces
each occurence with a given replacement string.

```hcl
replace(string, substring, replacement)
```

## Examples

```
> replace("1 + 2 + 3", "+", "-")
1 - 2 - 3
```
