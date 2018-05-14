---
layout: "functions"
page_title: "log function"
sidebar_current: "docs-funcs-numeric-log"
description: |-
  The log function returns the closest whole number less than or equal to
  the given value.
---

# `log` Function

`log` returns the the logarithm of a given number in a given base.

```hcl
log(number, base)
```

## Examples

```
> log(50, 10)
1.6989700043360185
> log(16, 2)
4
```

`log` and `ceil` can be used together to find the minimum number of binary
digits required to represent a given number of distinct values:

```
> ceil(log(15, 2))
4
> ceil(log(16, 2))
4
> ceil(log(17, 2))
5
```
