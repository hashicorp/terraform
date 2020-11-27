---
layout: "language"
page_title: "Function Calls - Configuration Language"
---

# Function Calls

The Terraform language has a number of
[built-in functions](/docs/configuration/functions.html) that can be used
in expressions to transform and combine values. These
are similar to the operators but all follow a common syntax:

```hcl
<FUNCTION NAME>(<ARGUMENT 1>, <ARGUMENT 2>)
```

The function name specifies which function to call. Each defined function
expects a specific number of arguments with specific value types, and returns a
specific value type as a result.

Some functions take an arbitrary number of arguments. For example, the `min`
function takes any amount of number arguments and returns the one that is
numerically smallest:

```hcl
min(55, 3453, 2)
```

A function call expression evaluates to the function's return value.

## Expanding Function Arguments

If the arguments to pass to a function are available in a list or tuple value,
that value can be _expanded_ into separate arguments. Provide the list value as
an argument and follow it with the `...` symbol:

```hcl
min([55, 2453, 2]...)
```

The expansion symbol is three periods (`...`), not a Unicode ellipsis character
(`…`). Expansion is a special syntax that is only available in function calls.

## Available Functions

For a full list of available functions, see
[the function reference](/docs/configuration/functions.html).

