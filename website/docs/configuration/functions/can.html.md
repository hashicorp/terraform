---
layout: "functions"
page_title: "can - Functions - Configuration Language"
sidebar_current: "docs-funcs-conversion-can"
description: |-
  The can function tries to evaluate an expression given as an argument and
  indicates whether the evaluation succeeded.
---

# `can` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`can` evaluates the given expression and returns a boolean value indicating
whether the expression produced a result without any errors.

This is a special function that is able to catch errors produced when evaluating
its argument. This function should be used with care, considering many of the
same caveats that apply to [`try`](./try.html), to avoid writing configurations
that are hard to read and maintain.

For most situations it's better to use [`try`](./try.html), because it allows
for more concise definition of fallback values for failing expressions.

The `can` function can only catch and handle _dynamic_ errors resulting from
access to data that isn't known until runtime. It will not catch errors
relating to expressions that can be proven to be invalid for any input, such
as a malformed resource reference.

~> **Warning:** The `can` function is intended only for concise testing of the
presence of and types of object attributes. Although it can technically accept
any sort of expression, we recommend using it only with simple attribute
references and type conversion functions as shown in the [`try`](./try.html)
examples. Overuse of `can` to suppress errors will lead to a configuration that
is hard to understand and maintain.

## Examples

```
> local.foo
{
  "bar" = "baz"
}
> can(local.foo.bar)
true
> can(local.foo.boop)
false
```

The `can` function will _not_ catch errors relating to constructs that are
provably invalid even before dynamic expression evaluation, such as a malformed
reference or a reference to a top-level object that has not been declared:

```
> can(local.nonexist)

Error: Reference to undeclared local value

A local value with the name "nonexist" has not been declared.
```

## Related Functions

* [`try`](./try.html), which tries evaluating a sequence of expressions and
  returns the result of the first one that succeeds.
