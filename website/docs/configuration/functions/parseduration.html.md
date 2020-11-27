---
layout: "language"
page_title: "parseduration - Functions - Configuration Language"
sidebar_current: "docs-funcs-datetime-parseduration"
description: |-
  The parseduration function parses the given string and return the result expressed in the given unit.
---

# `parseduration` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`parseduration` parses the given string and returns the result in the given
unit as an integer. The unit can be `"milliseconds"`, `"seconds"`, `"minutes"`
or `"hours"` and defaults to `"seconds"`.

If the given string is not a valid duration or the unit is invalid then
`parseduration` will produce an error.

## Examples

```
> parseduration("1m30s")
90

> parseduration("1.5s", "milliseconds")
1500

> parseduration("-1.5h", "seconds")
-5400

> parseduration("2h45m", "minutes")
165

> parseduration("aA", "hours")

Error: Invalid function argument

Invalid value for "duration" parameter: time: invalid duration "aA".
```
