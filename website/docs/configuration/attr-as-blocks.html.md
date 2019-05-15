---
layout: "docs"
page_title: "Attributes as Blocks - Configuration Language"
sidebar_current: "docs-config-attr-as-blocks"
description: |-
  For historical reasons, certain arguments within resource blocks can use either
  block or attribute syntax.
---

# Attributes as Blocks

-> **Note:** This page is an appendix to the Terraform documentation, and is
outside the normal navigation hierarchy. Most users do not need to know the full
details of the behavior described below.

## Summary

Many resource types use repeatable nested blocks to manage collections of
sub-objects related to the primary resource.

Rarely, some resource types _also_ support an argument with the same name as a
nested block type, and will purge any sub-objects of that type if that argument
is set to an empty list (`<ATTR> = []`).

Most users do not need to know any further details of this "nested block or
empty list" behavior. However, read further if you need to:

- Use Terraform's [JSON syntax](/docs/configuration/syntax-json.html) with this
  type of resource.
- Create a reusable module that wraps this type of resource.

## Details

In Terraform v0.12 and later, the language makes a distinction between
[argument syntax and nested block syntax](/docs/configuration/syntax.html#arguments-and-blocks)
within blocks:

* Argument syntax sets a named argument for the containing object. If the
  attribute has a default value then an explicitly-specified value entirely
  overrides that default.

* Nested block syntax represents a related child object of the container that
  has its own set of arguments. Where multiple such objects are possible, multiple
  blocks of the same type can be present. If the nested attributes themselves
  have default values, they are honored for each nested block separately,
  merging in with any explicitly-defined arguments.

The distinction between these is particularly important for
[JSON syntax](/docs/configuration/syntax-json.html)
because the same primitive JSON constructs (lists and objects) will be
interpreted differently depending on whether a particular name is an argument
or a nested block type.

However, in some cases existing provider features were relying on the
conflation of these two concepts in the language of Terraform v0.11 and earlier,
using nested block syntax in most cases but using argument syntax to represent
explicitly the idea of removing all existing objects of that type, since the
absense of any blocks was interpreted as "ignore any existing objects".

The information on this page only applies to certain special arguments that
were relying on this usage pattern prior to Terraform v0.12. The documentation
for each of those features links to this page for details of the special usage
patterns that apply. In all other cases, use either argument or nested block
syntax as directed by the examples in the documentation for a particular
resource type.

## Defining a Fixed Object Collection Value

When working with resource type arguments that behave in this way, it is valid
and we recommend to use the nested block syntax whenever defining a fixed
collection of objects:

```hcl
example {
  foo = "bar"
}
example {
  foo = "baz"
}
```

The above implicitly specifies a two-element list of objects assigned to the
`example` argument, treating it as if it were a nested block type.

If you need to explicitly call for zero `example` objects, you must use the
argument syntax with an empty list:

```hcl
example = []
```

These two forms cannot be mixed; there cannot be both explicitly zero `example`
objects and explicit single `example` blocks declared at the same time.

For true nested blocks where this special behavior does not apply, assigning
`[]` using argument syntax is not valid. The normal way to specify zero objects
of a type is to write no nested blocks at all.

## Arbitrary Expressions with Argument Syntax

Although we recommend using block syntax for simple cases for readability, the
names that work in this mode _are_ defined as arguments, and so it is possible
to use argument syntax to assign arbitrary dynamic expressions to them, as
long as the expression has the expected result type:

```hcl
example = [
  for name in var.names: {
    foo = name
  }
]
```

```hcl
# Not recommended, but valid: a constant list-of-objects expression
example = [
  {
    foo = "bar"
  },
  {
    foo = "baz"
  },
]
```

Because of the rule that argument declarations like this fully override any
default value, when creating a list-of-objects expression directly the usual
handling of optional arguments does not apply, so all of the arguments must be
assigned a value, even if it's an explicit `null`:

```hcl
example = [
  {
    # Cannot omit foo in this case, even though it would be optional in the
    # nested block syntax.
    foo = null
  },
]
```

If you are writing a reusable module that allows callers to pass in a list of
objects to assign to such an argument, you may wish to use the `merge` function
to populate any attributes the user didn't explicitly set, in order to give
the module user the effect of optional arguments:

```hcl
example = [
  for ex in var.examples: merge({
    foo = null # (or any other suitable default value)
  }, ex)
]
```

For the arguments that use the attributes-as-blocks usage mode, the above is
a better pattern than using
[`dynamic` blocks](/docs/configuration/expressions.html#dynamic-blocks)
because the case where the
caller provides an empty list will result in explicitly assigning an empty
list value, rather than assigning no value at all and thus retaining and
ignoring any existing objects. `dynamic` blocks are required for
dynamically-generating _normal_ nested blocks, though.

## In JSON syntax

Arguments that use this special mode are specified in JSON syntax always using
the [JSON expression mapping](/docs/configuration/syntax-json.html#expression-mapping)
to produce a list of objects.

The interpretation of these values in JSON syntax is, therefore, equivalent
to that described under _Arbitrary Expressions with Argument Syntax_ above,
but expressed in JSON syntax instead.

Due to the ambiguity of the JSON syntax, there is no way to distinguish based
on the input alone between argument and nested block usage, so the JSON syntax
cannot support the nested block processing mode for these arguments. This is,
unfortunately, one necessary concession on the equivalence between native syntax
and JSON syntax made pragmatically for compatibility with existing provider
design patterns. Providers may phase out such patterns in future major releases.
