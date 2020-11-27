---
layout: "language"
page_title: "Splat Expressions - Configuration Language"
---

# Splat Expressions

A _splat expression_ provides a more concise way to express a common
operation that could otherwise be performed with a `for` expression.

If `var.list` is a list of objects that all have an attribute `id`, then
a list of the ids could be produced with the following `for` expression:

```hcl
[for o in var.list : o.id]
```

This is equivalent to the following _splat expression:_

```hcl
var.list[*].id
```

The special `[*]` symbol iterates over all of the elements of the list given
to its left and accesses from each one the attribute name given on its
right. A splat expression can also be used to access attributes and indexes
from lists of complex types by extending the sequence of operations to the
right of the symbol:

```hcl
var.list[*].interfaces[0].name
```

The above expression is equivalent to the following `for` expression:

```hcl
[for o in var.list : o.interfaces[0].name]
```

Splat expressions are for lists only (and thus cannot be used [to reference resources
created with `for_each`](/docs/configuration/meta-arguments/for_each.html#referring-to-instances),
which are represented as maps in Terraform). However, if a splat expression is applied
to a value that is _not_ a list or tuple then the value is automatically wrapped in
a single-element list before processing.

For example, `var.single_object[*].id` is equivalent to `[var.single_object][*].id`,
or effectively `[var.single_object.id]`. This behavior is not interesting in most cases,
but it is particularly useful when referring to resources that may or may
not have `count` set, and thus may or may not produce a tuple value:

```hcl
aws_instance.example[*].id
```

The above will produce a list of ids whether `aws_instance.example` has
`count` set or not, avoiding the need to revise various other expressions
in the configuration when a particular resource switches to and from
having `count` set.

## Legacy (Attribute-only) Splat Expressions

An older variant of the splat expression is available for compatibility with
code written in older versions of the Terraform language. This is a less useful
version of the splat expression, and should be avoided in new configurations.

An "attribute-only" splat expression is indicated by the sequence `.*` (instead
of `[*]`):

```
var.list.*.interfaces[0].name
```

This form has a subtly different behavior, equivalent to the following
`for` expression:

```
[for o in var.list : o.interfaces][0].name
```

Notice that with the attribute-only splat expression the index operation
`[0]` is applied to the result of the iteration, rather than as part of
the iteration itself.
