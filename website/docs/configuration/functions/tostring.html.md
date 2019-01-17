---
layout: "functions"
page_title: "tostring - Functions - Configuration Language"
sidebar_current: "docs-funcs-conversion-tostring"
description: |-
  The tostring function converts a value to a string.
---

# `tostring` Function

`tostring` converts its argument to a string value.

Explicit type conversions are rarely necessary in Terraform because it will
convert types automatically where required. Use the explicit type conversion
functions only to normalize types returned in module outputs.

Only the primitive types (string, number, and bool) can be converted to string.
All other values will produce an error.

## Examples

```
> tostring("hello")
hello
> tostring(1)
1
> tostring(true)
true
> tostring([])
Error: Invalid function argument

Invalid value for "v" parameter: cannot convert tuple to string.
```
