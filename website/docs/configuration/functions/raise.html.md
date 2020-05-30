---
layout: "functions"
page_title: "can - Functions - Configuration Language"
sidebar_current: "docs-funcs-conversion-can"
description: |-
  The can function tries to evaluate an expression given as an argument and
  indicates whether the evaluation succeeded.
---

# `try` Function

-> **Note:** This page is about Terraform [TK - TODO: add version] and later.

`raise` produces an error message and stops execution. The `raise function can be placed in
code paths which will only evaluate in cases where execution should be blocked.

This is a special function that is able to raise errors whenever the function is executed.
Most common uses of `raise` include:

1. As the second argument to the `try()` function:
   `my_ratio = try(numerator / divisor, raise("An error occurred while calculating `my_ratio`. Did you try to use a negative divisor?"))`
2. In a conditional block:
   - `my_ratio = divisor > 0 ? numerator / divisor : raise("An error occurred while calculating `my_ratio`. Did you try to use a negative divisor?")`
3. As the final argument to the `coalesce()` function:
   - `my_ratio = numerator / coalesce(divisor, default_divisor, raise("Error. No non-null divisor found"))`

## Related Functions

* [`try`](./try.html), which tries evaluating a sequence of expressions and
  returns the result of the first one that succeeds.
* [`can`](./can.html), which tries evaluating an expression and returns a
  boolean value indicating whether it succeeded.
