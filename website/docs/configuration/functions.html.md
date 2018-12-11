---
layout: "functions"
page_title: "Configuration Functions"
sidebar_current: "docs-config-functions"
description: |-
  The Terraform language has a number of built-in functions that can be called
  from within expressions to transform and combine values.
---

# Built-in Functions

The Terraform language includes a number of built-in functions that you can
call from within expressions to transform and combine values. The general
syntax for function calls is a function name followed by comma-separated
arguments in parentheses:

```hcl
max(5, 12, 9)
```

For more details on syntax, see
[_Function Calls_](./expressions.html#function-calls)
on the Expressions page.

The Terraform language does not support user-defined functions, and so only
the functions built in to the language are available for use. The navigation
for this section includes a list of all of the available built-in functions.

You can experiment with the behavior of Terraform's built-in functions from
the Terraform expression console, by running
[the `terraform console` command](/docs/commands/console.html):

```
> max(5, 12, 9)
12
```

The examples in the documentation for each function use console output to
illustrate the result of calling the function with different parameters.
