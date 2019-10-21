---
layout: "functions"
page_title: "tonumber - Functions - Configuration Language"
sidebar_current: "docs-funcs-conversion-tonumber"
description: |-
  The tonumber function converts a value to a number.
---

# `tonumber` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`tonumber` converts its argument to a number value.

Explicit type conversions are rarely necessary in Terraform because it will
convert types automatically where required. Use the explicit type conversion
functions only to normalize types returned in module outputs.

Only numbers and strings containing decimal representations of numbers can be
converted to number. All other values will produce an error.

## Examples

```
> tonumber(1)
1
> tonumber("1")
1
> tonumber("no")
Error: Invalid function argument

Invalid value for "v" parameter: cannot convert "no" to number: string must be
a decimal representation of a number.
```
