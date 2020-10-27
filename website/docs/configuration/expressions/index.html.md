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
Each language feature's documentation describes any restrictions it places on expressions.

You can experiment with the behavior of Terraform's expressions from
the Terraform expression console, by running
[the `terraform console` command](/docs/commands/console.html).

The other pages in this section describe the features of Terraform's
expression syntax.
