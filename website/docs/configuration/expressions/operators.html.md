---
layout: "language"
page_title: "Operators - Configuration Language"
---

# Arithmetic and Logical Operators

An _operator_ is a type of expression that transforms or combines one or more
other expressions. Operators either combine two values in some way to
produce a third result value, or transform a single given value to
produce a single result.

Operators that work on two values place an operator symbol between the two
values, similar to mathematical notation: `1 + 2`. Operators that work on
only one value place an operator symbol before that value, like
`!true`.

The Terraform language has a set of operators for both arithmetic and logic,
which are similar to operators in programming languages such as JavaScript
or Ruby.

When multiple operators are used together in an expression, they are evaluated
in the following order of operations:

1. `!`, `-` (multiplication by `-1`)
1. `*`, `/`, `%`
1. `+`, `-` (subtraction)
1. `>`, `>=`, `<`, `<=`
1. `==`, `!=`
1. `&&`
1. `||`

Parentheses can be used to override the default order of operations. Without
parentheses, higher levels are evaluated first, so `1 + 2 * 3` is interpreted
as `1 + (2 * 3)` and _not_ as `(1 + 2) * 3`.

The different operators can be gathered into a few different groups with
similar behavior, as described below. Each group of operators expects its
given values to be of a particular type. Terraform will attempt to convert
values to the required type automatically, or will produce an error message
if this automatic conversion is not possible.

## Arithmetic Operators

The arithmetic operators all expect number values and produce number values
as results:

* `a + b` returns the result of adding `a` and `b` together.
* `a - b` returns the result of subtracting `b` from `a`.
* `a * b` returns the result of multiplying `a` and `b`.
* `a / b` returns the result of dividing `a` by `b`.
* `a % b` returns the remainder of dividing `a` by `b`. This operator is
  generally useful only when used with whole numbers.
* `-a` returns the result of multiplying `a` by `-1`.

## Equality Operators

The equality operators both take two values of any type and produce boolean
values as results.

* `a == b` returns `true` if `a` and `b` both have the same type and the same
  value, or `false` otherwise.
* `a != b` is the opposite of `a == b`.

## Comparison Operators

The comparison operators all expect number values and produce boolean values
as results.

* `a < b` returns `true` if `a` is less than `b`, or `false` otherwise.
* `a <= b` returns `true` if `a` is less than or equal to `b`, or `false`
  otherwise.
* `a > b` returns `true` if `a` is greater than `b`, or `false` otherwise.
* `a >= b` returns `true` if `a` is greater than or equal to `b`, or `false` otherwise.

## Logical Operators

The logical operators all expect bool values and produce bool values as results.

* `a || b` returns `true` if either `a` or `b` is `true`, or `false` if both are `false`.
* `a && b` returns `true` if both `a` and `b` are `true`, or `false` if either one is `false`.
* `!a` returns `true` if `a` is `false`, and `false` if `a` is `true`.
