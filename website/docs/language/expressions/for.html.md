---
layout: "language"
page_title: "For Expressions - Configuration Language"
---

# `for` Expressions

A _`for` expression_ creates a complex type value by transforming
another complex type value. Each element in the input value
can correspond to either one or zero values in the result, and an arbitrary
expression can be used to transform each input element into an output element.

For example, if `var.list` is a list of strings, then the following expression
produces a list of strings with all-uppercase letters:

```hcl
[for s in var.list : upper(s)]
```

This `for` expression iterates over each element of `var.list`, and then
evaluates the expression `upper(s)` with `s` set to each respective element.
It then builds a new tuple value with all of the results of executing that
expression in the same order.

The type of brackets around the `for` expression decide what type of result
it produces. The above example uses `[` and `]`, which produces a tuple. If
`{` and `}` are used instead, the result is an object, and two result
expressions must be provided separated by the `=>` symbol:

```hcl
{for s in var.list : s => upper(s)}
```

This expression produces an object whose attributes are the original elements
from `var.list` and their corresponding values are the uppercase versions.

A `for` expression can also include an optional `if` clause to filter elements
from the source collection, which can produce a value with fewer elements than
the source:

```
[for s in var.list : upper(s) if s != ""]
```

The source value can also be an object or map value, in which case two
temporary variable names can be provided to access the keys and values
respectively:

```
[for k, v in var.map : length(k) + length(v)]
```

Finally, if the result type is an object (using `{` and `}` delimiters) then
the value result expression can be followed by the `...` symbol to group
together results that have a common key:

```
{for s in var.list : substr(s, 0, 1) => s... if s != ""}
```

For expressions are particularly useful when combined with other language
features to combine collections together in various ways. For example,
the following two patterns are commonly used when constructing map values
to use with
[the `for_each` meta-argument](/docs/configuration/meta-arguments/for_each.html):

* Transform a multi-level nested structure into a flat list by
  [using nested `for` expressions with the `flatten` function](/docs/configuration/functions/flatten.html#flattening-nested-structures-for-for_each).
* Produce an exhaustive list of combinations of elements from two or more
  collections by
  [using the `setproduct` function inside a `for` expression](/docs/configuration/functions/setproduct.html#finding-combinations-for-for_each).
