---
layout: "language"
page_title: "Conditional Expressions - Configuration Language"
---

# Conditional Expressions

> **Hands-on:** Try the [Create Dynamic Expressions](https://learn.hashicorp.com/tutorials/terraform/expressions?in=terraform/configuration-language&utm_source=WEBSITE&utm_medium=WEB_IO&utm_offer=ARTICLE_PAGE&utm_content=DOCS) tutorial on HashiCorp Learn.

A _conditional expression_ uses the value of a bool expression to select one of
two values.

The syntax of a conditional expression is as follows:

```hcl
condition ? true_val : false_val
```

If `condition` is `true` then the result is `true_val`. If `condition` is
`false` then the result is `false_val`.

A common use of conditional expressions is to define defaults to replace
invalid values:

```
var.a != "" ? var.a : "default-a"
```

If `var.a` is an empty string then the result is `"default-a"`, but otherwise
it is the actual value of `var.a`.

## Conditions

The condition can be any expression that resolves to a boolean value. This will
usually be an expression that uses the equality, comparison, or logical
operators.

## Result Types

The two result values may be of any type, but they must both
be of the _same_ type so that Terraform can determine what type the whole
conditional expression will return without knowing the condition value.

