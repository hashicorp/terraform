---
layout: "functions"
page_title: "range - Functions - Configuration Language"
sidebar_current: "docs-funcs-collection-range"
description: |-
  The range function generates sequences of numbers.
---

# `range` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`range` generates a list of numbers using a start value, a limit value,
and a step value.

```hcl
range(max)
range(start, limit)
range(start, limit, step)
```

The `start` and `step` arguments can be omitted, in which case `start` defaults
to zero and `step` defaults to either one or negative one depending on whether
`limit` is greater than or less than `start`.

The resulting list is created by starting with the given `start` value and
repeatedly adding `step` to it until the result is equal to or beyond `limit`.

The interpretation of `limit` depends on the direction of `step`: for a positive
step, the sequence is complete when the next number is greater than or equal
to `limit`. For a negative step, it's complete when less than or equal.

The sequence-building algorithm follows the following pseudocode:

```
let num = start
while num <= limit: (or, for negative step, num >= limit)
  append num to the sequence
  num = num + step
return the sequence
```

Because the sequence is created as a physical list in memory, Terraform imposes
an artificial limit of 1024 numbers in the resulting sequence in order to avoid
unbounded memory usage if, for example, a very large value were accidentally
passed as the limit or a very small value as the step. If the algorithm above
would append the 1025th number to the sequence, the function immediately exits
with an error.

We recommend iterating over existing collections where possible, rather than
creating ranges. However, creating small numerical sequences can sometimes
be useful when combined with other collections in collection-manipulation
functions or `for` expressions.

## Examples

```
> range(3)
[
  0,
  1,
  2,
]

> range(1, 4)
[
  1,
  2,
  3,
]

> range(1, 8, 2)
[
  1,
  3,
  5,
  7,
]

> range(1, 4, 0.5)
[
  1,
  1.5,
  2,
  2.5,
  3,
  3.5,
]

> range(4, 1)
[
  4,
  3,
  2,
]

> range(10, 5, -2)
[
  10,
  8,
  6,
]
```

The `range` function is primarily useful when working with other collections
to produce a certain number of instances of something. For example:

```hcl
variable "name_counts" {
  type    = map(number)
  default = {
    "foo" = 2
    "bar" = 4
  }
}

locals {
  expanded_names = {
    for name, count in var.name_counts : name => [
      for i in range(count) : format("%s%02d", name, i)
    ]
  }
}

output "expanded_names" {
  value = local.expanded_names
}

# Produces the following expanded_names value when run with the default
# "name_counts":
#
# {
#   "bar" = [
#     "bar00",
#     "bar01",
#     "bar02",
#     "bar03",
#   ]
#   "foo" = [
#     "foo00",
#     "foo01",
#   ]
# }
```
