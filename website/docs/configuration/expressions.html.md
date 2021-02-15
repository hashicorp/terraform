---
layout: "language"
page_title: "Expressions Landing Page - Configuration Language"
sidebar_current: "docs-config-expressions"
---

# Expressions Landing Page

To improve navigation, we've split the old Expressions page into several smaller
pages.

<a id="types-and-values"></a>
<a id="advanced-type-details"></a>
<a id="type-conversion"></a>
<a id="literal-expressions"></a>
<a id="indices-and-attributes"></a>

## Types and Values, Literal Expressions, Indices and Attributes

Terraform's types are `string`, `number`, `bool`, `list`, `tuple`, `map`,
`object`, and `null`.

This information has moved to
[Types and Values](/docs/language/expressions/types.html).

<div style="height: 100vh; margin: 0; padding: 0;"></div>



<a id="references-to-named-values"></a>
<a id="local-named-values"></a>
<a id="named-values-and-dependencies"></a>
<a id="references-to-resource-attributes"></a>
<a id="local-named-values-1"></a>
<a id="values-not-yet-known"></a>

## References to Named Values (Resource Attributes, Variables, etc.)

You can refer to certain values by name, like `var.some_variable` or
`aws_instance.example.ami`.

This information has moved to
[References to Values](/docs/language/expressions/references.html).

<div style="height: 100vh; margin: 0; padding: 0;"></div>



<a id="arithmetic-operators"></a>
<a id="equality-operators"></a>
<a id="comparison-operators"></a>
<a id="logical-operators"></a>

## Arithmetic and Logical Operators

Operators are expressions that transform other expressions, like adding two
numbers (`+`) or comparing two values to get a bool (`==`, `>=`, etc.).

This information has moved to
[Operators](/docs/language/expressions/references.html).

<div style="height: 100vh; margin: 0; padding: 0;"></div>



## Conditional Expressions

The `condition ? true_val : false_val` expression chooses between two
expressions based on a bool condition.

This information has moved to
[Conditional Expressions](/docs/language/expressions/conditionals.html).

<div style="height: 100vh; margin: 0; padding: 0;"></div>



<a id="expanding-function-arguments"></a>
<a id="available-functions"></a>

## Function Calls

Terraform's functions can be called like `function_name(arg1, arg2)`.

This information has moved to
[Function Calls](/docs/language/expressions/function-calls.html).

<div style="height: 100vh; margin: 0; padding: 0;"></div>



<a id="for-expressions"></a>

## `for` Expressions

Expressions like `[for s in var.list : upper(s)]` can transform a complex type
value into another complex type value.

This information has moved to
[For Expressions](/docs/language/expressions/for.html).

<div style="height: 100vh; margin: 0; padding: 0;"></div>



<a id="splat-expressions"></a>
<a id="legacy-attribute-only-splat-expressions"></a>

## Splat Expressions

Expressions like `var.list[*].id` can extract simpler collections from complex
collections.

This information has moved to
[Splat Expressions](/docs/language/expressions/splat.html).

<div style="height: 100vh; margin: 0; padding: 0;"></div>



<a id="dynamic-blocks"></a>
<a id="best-practices-for-dynamic-blocks"></a>

## `dynamic` Blocks

The special `dynamic` block type serves the same purpose as a `for` expression,
except it creates multiple repeatable nested blocks instead of a complex value.

This information has moved to
[Dynamic Blocks](/docs/language/expressions/dynamic-blocks.html).

<div style="height: 100vh; margin: 0; padding: 0;"></div>



<a id="string-literals"></a>
<a id="string-templates"></a>
<a id="interpolation"></a>
<a id="directives"></a>

## String Literals and String Templates

Strings can be `"double-quoted"` or

```hcl
<<EOT
heredocs
EOT
```

Strings can also include escape sequences like `\n`, interpolation sequences
(`${ ... }`), and template sequences (`%{ ... }`).

This information has moved to
[Strings and Templates](/docs/language/expressions/strings.html).

<div style="height: 100vh; margin: 0; padding: 0;"></div>
