---
layout: "language"
page_title: "Dynamic Blocks - Configuration Language"
---


# `dynamic` Blocks

Within top-level block constructs like resources, expressions can usually be
used only when assigning a value to an argument using the `name = expression`
form. This covers many uses, but some resource types include repeatable _nested
blocks_ in their arguments, which do not accept expressions:

```hcl
resource "aws_elastic_beanstalk_environment" "tfenvtest" {
  name = "tf-test-name" # can use expressions here

  setting {
    # but the "setting" block is always a literal block
  }
}
```

You can dynamically construct repeatable nested blocks like `setting` using a
special `dynamic` block type, which is supported inside `resource`, `data`,
`provider`, and `provisioner` blocks:

```hcl
resource "aws_elastic_beanstalk_environment" "tfenvtest" {
  name                = "tf-test-name"
  application         = "${aws_elastic_beanstalk_application.tftest.name}"
  solution_stack_name = "64bit Amazon Linux 2018.03 v2.11.4 running Go 1.12.6"

  dynamic "setting" {
    for_each = var.settings
    content {
      namespace = setting.value["namespace"]
      name = setting.value["name"]
      value = setting.value["value"]
    }
  }
}
```

A `dynamic` block acts much like a `for` expression, but produces nested blocks
instead of a complex typed value. It iterates over a given complex value, and
generates a nested block for each element of that complex value.

- The label of the dynamic block (`"setting"` in the example above) specifies
  what kind of nested block to generate.
- The `for_each` argument provides the complex value to iterate over.
- The `iterator` argument (optional) sets the name of a temporary variable
  that represents the current element of the complex value. If omitted, the name
  of the variable defaults to the label of the `dynamic` block (`"setting"` in
  the example above).
- The `labels` argument (optional) is a list of strings that specifies the block
  labels, in order, to use for each generated block. You can use the temporary
  iterator variable in this value.
- The nested `content` block defines the body of each generated block. You can
  use the temporary iterator variable inside this block.

Since the `for_each` argument accepts any collection or structural value,
you can use a `for` expression or splat expression to transform an existing
collection.

The iterator object (`setting` in the example above) has two attributes:

* `key` is the map key or list element index for the current element. If the
  `for_each` expression produces a _set_ value then `key` is identical to
  `value` and should not be used.
* `value` is the value of the current element.

A `dynamic` block can only generate arguments that belong to the resource type,
data source, provider or provisioner being configured. It is _not_ possible
to generate meta-argument blocks such as `lifecycle` and `provisioner`
blocks, since Terraform must process these before it is safe to evaluate
expressions.

The `for_each` value must be a map or set with one element per desired
nested block. If you need to declare resource instances based on a nested
data structure or combinations of elements from multiple data structures you
can use Terraform expressions and functions to derive a suitable value.
For some common examples of such situations, see the
[`flatten`](/docs/configuration/functions/flatten.html)
and
[`setproduct`](/docs/configuration/functions/setproduct.html)
functions.

## Best Practices for `dynamic` Blocks

Overuse of `dynamic` blocks can make configuration hard to read and maintain, so
we recommend using them only when you need to hide details in order to build a
clean user interface for a re-usable module. Always write nested blocks out
literally where possible.

