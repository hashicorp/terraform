---
layout: "language"
page_title: "Expressions - Configuration Language"
description: "An overview of expressions to reference or compute values in Terraform configurations, including types, operators, and functions."
---

# Expressions

> **Hands-on:** Try the [Create Dynamic Expressions](https://learn.hashicorp.com/tutorials/terraform/expressions?in=terraform/configuration-language&utm_source=WEBSITE&utm_medium=WEB_IO&utm_offer=ARTICLE_PAGE&utm_content=DOCS) tutorial on HashiCorp Learn.

_Expressions_ are used to refer to or compute values within a configuration.
The simplest expressions are just literal values, like `"hello"` or `5`,
but the Terraform language also allows more complex expressions such as
references to data exported by resources, arithmetic, conditional evaluation,
and a number of built-in functions.

Expressions can be used in a number of places in the Terraform language,
but some contexts limit which expression constructs are allowed,
such as requiring a literal value of a particular type or forbidding
[references to resource attributes](/docs/language/expressions/references.html#references-to-resource-attributes).
Each language feature's documentation describes any restrictions it places on
expressions.

You can experiment with the behavior of Terraform's expressions from
the Terraform expression console, by running
[the `terraform console` command](/docs/cli/commands/console.html).

The other pages in this section describe the features of Terraform's
expression syntax.

- [Types and Values](/docs/language/expressions/types.html)
  documents the data types that Terraform expressions can resolve to, and the
  literal syntaxes for values of those types.

- [Strings and Templates](/docs/language/expressions/strings.html)
  documents the syntaxes for string literals, including interpolation sequences
  and template directives.

- [References to Values](/docs/language/expressions/references.html)
  documents how to refer to named values like variables and resource attributes.

- [Operators](/docs/language/expressions/operators.html)
  documents the arithmetic, comparison, and logical operators.

- [Function Calls](/docs/language/expressions/function-calls.html)
  documents the syntax for calling Terraform's built-in functions.

- [Conditional Expressions](/docs/language/expressions/conditionals.html)
  documents the `<CONDITION> ? <TRUE VAL> : <FALSE VAL>` expression, which
  chooses between two values based on a bool condition.

- [For Expressions](/docs/language/expressions/for.html)
  documents expressions like `[for s in var.list : upper(s)]`, which can
  transform a complex type value into another complex type value.

- [Splat Expressions](/docs/language/expressions/splat.html)
  documents expressions like `var.list[*].id`, which can extract simpler
  collections from more complicated expressions.

- [Dynamic Blocks](/docs/language/expressions/dynamic-blocks.html)
  documents a way to create multiple repeatable nested blocks within a resource
  or other construct.

- [Type Constraints](/docs/language/expressions/type-constraints.html)
  documents the syntax for referring to a type, rather than a value of that
  type. Input variables expect this syntax in their `type` argument.

- [Version Constraints](/docs/language/expressions/version-constraints.html)
  documents the syntax of special strings that define a set of allowed software
  versions. Terraform uses version constraints in several places.
