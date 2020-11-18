---
layout: "language"
page_title: "Expressions - Configuration Language"
---

# Expressions

_Expressions_ are used to refer to or compute values within a configuration.
The simplest expressions are just literal values, like `"hello"` or `5`,
but the Terraform language also allows more complex expressions such as
references to data exported by resources, arithmetic, conditional evaluation,
and a number of built-in functions.

Expressions can be used in a number of places in the Terraform language,
but some contexts limit which expression constructs are allowed,
such as requiring a literal value of a particular type or forbidding
[references to resource attributes](/docs/configuration/expressions/references.html#references-to-resource-attributes).
Each language feature's documentation describes any restrictions it places on
expressions.

You can experiment with the behavior of Terraform's expressions from
the Terraform expression console, by running
[the `terraform console` command](/docs/commands/console.html).

The other pages in this section describe the features of Terraform's
expression syntax.

- [Types and Values](/docs/configuration/expressions/types.html)
  documents the data types that Terraform expressions can resolve to, and the
  literal syntaxes for values of those types.

- [Strings and Templates](/docs/configuration/expressions/strings.html)
  documents the syntaxes for string literals, including interpolation sequences
  and template directives.

- [References to Values](/docs/configuration/expressions/references.html)
  documents how to refer to named values like variables and resource attributes.

- [Operators](/docs/configuration/expressions/references.html)
  documents the arithmetic, comparison, and logical operators.

- [Function Calls](/docs/configuration/expressions/function-calls.html)
  documents the syntax for calling Terraform's built-in functions.

- [Conditional Expressions](/docs/configuration/expressions/conditionals.html)
  documents the `<CONDITION> ? <TRUE VAL> : <FALSE VAL>` expression, which
  chooses between two values based on a bool condition.

- [For Expressions](/docs/configuration/expressions/for.html)
  documents expressions like `[for s in var.list : upper(s)]`, which can
  transform a complex type value into another complex type value.

- [Splat Expressions](/docs/configuration/expressions/splat.html)
  documents expressions like `var.list[*].id`, which can extract simpler
  collections from more complicated expressions.

- [Dynamic Blocks](/docs/configuration/expressions/dynamic-blocks.html)
  documents a way to create multiple repeatable nested blocks within a resource
  or other construct.

- [Type Constraints](/docs/configuration/types.html)
  documents the syntax for referring to a type, rather than a value of that
  type. Input variables expect this syntax in their `type` argument.

- [Version Constraints](/docs/configuration/version-constraints.html)
  documents the syntax of special strings that define a set of allowed software
  versions. Terraform uses version constraints in several places.
