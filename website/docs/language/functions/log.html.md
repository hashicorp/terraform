---
layout: "language"
page_title: "log - Functions - Configuration Language"
sidebar_current: "docs-funcs-numeric-log"
description: |-
  The log function returns the logarithm of a given number in a given base.
---

# `log` Function

`log` returns the logarithm of a given number in a given base.

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

`log` and `floor` can be used together to find the minimum number of binary
digits required to represent a given number of distinct values:

```
> floor(log(15, 2)) + 1
4
> floor(log(16, 2)) + 1
5
> floor(log(17, 2)) + 1
5
```
