---
layout: "functions"
page_title: "tobool - Functions - Configuration Language"
sidebar_current: "docs-funcs-conversion-tobool"
description: |-
  The tobool function converts a value to boolean.
---

# `tobool` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`tobool` converts its argument to a boolean value.

Explicit type conversions are rarely necessary in Terraform because it will
convert types automatically where required. Use the explicit type conversion
functions only to normalize types returned in module outputs.

Only boolean values and the exact strings `"true"` and `"false"` can be
converted to boolean. All other values will produce an error.

## Examples

```
> tobool(true)
true
> tobool("true")
true
> tobool("no")
Error: Invalid function argument

Invalid value for "v" parameter: cannot convert "no" to bool: only the strings
"true" or "false" are allowed.

> tobool(1)
Error: Invalid function argument

Invalid value for "v" parameter: cannot convert number to bool.
```
