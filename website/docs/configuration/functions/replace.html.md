---
layout: "functions"
page_title: "replace - Functions - Configuration Language"
sidebar_current: "docs-funcs-string-replace"
description: |-
  The replace function searches a given string for another given substring,
  and replaces all occurrences with a given replacement string.
---

# `replace` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`replace` searches a given string for another given substring, and replaces
each occurrence with a given replacement string.

```hcl
replace(string, substring, replacement)
```

## Examples

```
> replace("1 + 2 + 3", "+", "-")
1 - 2 - 3
```
