---
layout: "docs"
page_title: "Expressions - Configuration Language"
sidebar_current: "docs-config-expressions"
description: |-
  The Terraform language allows the use of expressions to access data exported
  by resources and to transform and combine that data to produce other values.
---

# Expressions

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../configuration-0-11/interpolation.html).

_Expressions_ are used to refer to or compute values within a configuration.
The simplest expressions are just literal values, like `"hello"` or `5`,
but the Terraform language also allows more complex expressions such as
references to data exported by resources, arithmetic, conditional evaluation,
and a number of built-in functions.

Expressions can be used in a number of places in the Terraform language,
but some contexts limit which expression constructs are allowed,
such as requiring a literal value of a particular type or forbidding
references to resource attributes. Each language feature's documentation
describes any restrictions it places on expressions.

You can experiment with the behavior of Terraform's expressions from
the Terraform expression console, by running
[the `terraform console` command](/docs/commands/console.html).

The rest of this page describes all of the features of Terraform's
expression syntax.

## Types and Values

The result of an expression is a _value_. All values have a _type_, which
dictates where that value can be used and what transformations can be
applied to it.

The Terraform language uses the following types for its values:

* `string`: a sequence of Unicode characters representing some text, like
  `"hello"`.
* `number`: a numeric value. The `number` type can represent both whole
  numbers like `15` and fractional values like `6.283185`.
* `bool`: either `true` or `false`. `bool` values can be used in conditional
  logic.
* `list` (or `tuple`): a sequence of values, like
  `["us-west-1a", "us-west-1c"]`. Elements in a list or tuple are identified by
  consecutive whole numbers, starting with zero.
* `map` (or `object`): a group of values identified by named labels, like
  `{name = "Mabel", age = 52}`.

Strings, numbers, and bools are sometimes called _primitive types._ Lists/tuples and maps/objects are sometimes called _complex types,_ _structural types,_ or _collection types._

Finally, there is one special value that has _no_ type:

* `null`: a value that represents _absence_ or _omission._ If you set an
  argument of a resource or module to `null`, Terraform behaves as though you
  had completely omitted it — it will use the argument's default value if it has
  one, or raise an error if the argument is mandatory. `null` is most useful in
  conditional expressions, so you can dynamically omit an argument if a
  condition isn't met.

### Advanced Type Details

In most situations, lists and tuples behave identically, as do maps and objects.
Whenever the distinction isn't relevant, the Terraform documentation uses each
pair of terms interchangeably (with a historical preference for "list" and
"map").

However, module authors and provider developers should understand the
differences between these similar types (and the related `set` type), since they
offer different ways to restrict the allowed values for input variables and
resource arguments.

For complete details about these types (and an explanation of why the difference
usually doesn't matter), see [Type Constraints](./types.html).

### Type Conversion

Expressions are most often used to set values for the arguments of resources and
child modules. In these cases, the argument has an expected type and the given
expression must produce a value of that type.

Where possible, Terraform automatically converts values from one type to
another in order to produce the expected type. If this isn't possible, Terraform
will produce a type mismatch error and you must update the configuration with a
more suitable expression.

Terraform automatically converts number and bool values to strings when needed.
It also converts strings to numbers or bools, as long as the string contains a
valid representation of a number or bool value.

* `true` converts to `"true"`, and vice-versa
* `false` converts to `"false"`, and vice-versa
* `15` converts to `"15"`, and vice-versa

## Literal Expressions

A _literal expression_ is an expression that directly represents a particular
constant value. Terraform has a literal expression syntax for each of the value
types described above:

* Strings are usually represented by a double-quoted sequence of Unicode
  characters, `"like this"`. There is also a "heredoc" syntax for more complex
  strings. String literals are the most complex kind of literal expression in
  Terraform, and have additional documentation on this page:
    * See [String Literals](#string-literals) below for information about escape
      sequences and the heredoc syntax.
    * See [String Templates](#string-templates) below for information about
      interpolation and template directives.
* Numbers are represented by unquoted sequences of digits with or without a
  decimal point, like `15` or `6.283185`.
* Bools are represented by the unquoted symbols `true` and `false`.
* The null value is represented by the unquoted symbol `null`.
* Lists/tuples are represented by a pair of square brackets containing a
  comma-separated sequence of values, like `["a", 15, true]`.

    List literals can be split into multiple lines for readability, but always
    require a comma between values. A comma after the final value is allowed,
    but not required. Values in a list can be arbitrary expressions.
* Maps/objects are represented by a pair of curly braces containing a series of
  `<KEY> = <VALUE>` pairs:

    ```hcl
    {
      name = "John"
      age  = 52
    }
    ```

    Key/value pairs can be separated by either a comma or a line break. Values
    can be arbitrary expressions. Keys are strings; they can be left unquoted if
    they are a valid [identifier](./syntax.html#identifiers), but must be quoted
    otherwise. You can use a non-literal expression as a key by wrapping it in
    parentheses, like `(var.business_unit_tag_name) = "SRE"`.

## Indices and Attributes

[inpage-index]: #indices-and-attributes

Elements of list/tuple and map/object values can be accessed using
the square-bracket index notation, like `local.list[3]`. The expression within
the brackets must be a whole number for list and tuple values or a string
for map and object values.

Map/object attributes with names that are valid identifiers can also be accessed
using the dot-separated attribute notation, like `local.object.attrname`.
In cases where a map might contain arbitrary user-specified keys, we recommend
using only the square-bracket index notation (`local.map["keyname"]`).

## References to Named Values

Terraform makes several kinds of named values available. Each of these names is
an expression that references the associated value; you can use them as
standalone expressions, or combine them with other expressions to compute new
values.

The following named values are available:

* `<RESOURCE TYPE>.<NAME>` is an object representing a
  [managed resource](./resources.html) of the given type
  and name. The attributes of the resource can be accessed using
  [dot or square bracket notation][inpage-index].

    Any named value that does not match another pattern listed below
    will be interpreted by Terraform as a reference to a managed resource.

    If the resource has the `count` argument set, the value of this expression
    is a _list_ of objects representing its instances.

    For more information, see
    [references to resource attributes](#references-to-resource-attributes) below.
* `var.<NAME>` is the value of the
  [input variable](./variables.html) of the given name.
* `local.<NAME>` is the value of the
  [local value](./locals.html) of the given name.
* `module.<MODULE NAME>.<OUTPUT NAME>` is the value of the specified
  [output value](./outputs.html) from a
  [child module](./modules.html) called by the current module.
* `data.<DATA TYPE>.<NAME>` is an object representing a
  [data resource](./data-sources.html) of the given data
  source type and name. If the resource has the `count` argument set, the value
  is a list of objects representing its instances.
* `path.module` is the filesystem path of the module where the expression
  is placed.
* `path.root` is the filesystem path of the root module of the configuration.
* `path.cwd` is the filesystem path of the current working directory. In
  normal use of Terraform this is the same as `path.root`, but some advanced
  uses of Terraform run it from a directory other than the root module
  directory, causing these paths to be different.
* `terraform.workspace` is the name of the currently selected
  [workspace](/docs/state/workspaces.html).

Although many of these names use dot-separated paths that resemble
[attribute notation][inpage-index] for elements of object values, they are not
implemented as real objects. This means you must use them exactly as written:
you cannot use square-bracket notation to replace the dot-separated paths, and
you cannot iterate over the "parent object" of a named entity (for example, you
cannot use `aws_instance` in a `for` expression).

### Named Values and Dependencies

Constructs like resources and module calls often use references to named values
in their block bodies, and Terraform analyzes these expressions to automatically
infer dependencies between objects. For example, an expression in a resource
argument that refers to another managed resource creates an implicit dependency
between the two resources.

### References to Resource Attributes

The most common reference type is a reference to an attribute of a resource
which has been declared either with a `resource` or `data` block. Because
the contents of such blocks can be quite complicated themselves, expressions
referring to these contents can also be complicated.

Consider the following example resource block:

```hcl
resource "aws_instance" "example" {
  ami           = "ami-abc123"
  instance_type = "t2.micro"

  ebs_block_device {
    device_name = "sda2"
    volume_size = 16
  }
  ebs_block_device {
    device_name = "sda3"
    volume_size = 20
  }
}
```

The documentation for [`aws_instance`](/docs/providers/aws/r/instance.html)
lists all of the arguments and nested blocks supported for this resource type,
and also lists a number of attributes that are _exported_ by this resource
type. All of these different resource type schema constructs are available
for use in references, as follows:

* The `ami` argument set in the configuration can be used elsewhere with
  the reference expression `aws_instance.example.ami`.
* The `id` attribute exported by this resource type can be read using the
  same syntax, giving `aws_instance.example.id`.
* The arguments of the `ebs_block_device` nested blocks can be accessed using
  a [splat expression](#splat-expressions). For example, to obtain a list of
  all of the `device_name` values, use
  `aws_instance.example.ebs_block_device[*].device_name`.
* The nested blocks in this particular resource type do not have any exported
  attributes, but if `ebs_block_device` were to have a documented `id`
  attribute then a list of them could be accessed similarly as
  `aws_instance.example.ebs_block_device[*].id`.
* Sometimes nested blocks are defined as taking a logical key to identify each
  block, which serves a similar purpose as the resource's own name by providing
  a convenient way to refer to that single block in expressions. If `aws_instance`
  had a hypothetical nested block type `device` that accepted such a key, it
  would look like this in configuration:

  ```hcl
    device "foo" {
      size = 2
    }
    device "bar" {
      size = 4
    }
  ```

  Arguments inside blocks with _keys_ can be accessed using index syntax, such
  as `aws_instance.example.device["foo"].size`.

  To obtain a map of values of a particular argument for _labelled_ nested
  block types, use a [`for` expression](#for-expressions):
  `[for k, device in aws_instance.example.device : k => device.size]`.

When a particular resource has the special
[`count`](https://www.terraform.io/docs/configuration/resources.html#count-multiple-resource-instances)
argument set, the resource itself becomes a list of instance objects rather than
a single object. In that case, access the attributes of the instances using
either [splat expressions](#splat-expressions) or index syntax:

* `aws_instance.example[*].id` returns a list of all of the ids of each of the
  instances.
* `aws_instance.example[0].id` returns just the id of the first instance.

### Local Named Values

Within the bodies of certain expressions, or in some other specific contexts,
there are other named values available beyond the global values listed above.
(For example, the body of a resource block where `count` is set can use a
special `count.index` value.) These local names are described in the
documentation for the specific contexts where they appear.

-> **Note:** Local named values are often referred to as _variables_ or
_temporary variables_ in their documentation. These are not [input
variables](./variables.html); they are just arbitrary names
that temporarily represent a value.

### Values Not Yet Known

When Terraform is planning a set of changes that will apply your configuration,
some resource attribute values cannot be populated immediately because their
values are decided dynamically by the remote system. For example, if a
particular remote object type is assigned a generated unique id on creation,
Terraform cannot predict the value of this id until the object has been created.

To allow expressions to still be evaluated during the plan phase, Terraform
uses special "unknown value" placeholders for these results. In most cases you
don't need to do anything special to deal with these, since the Terraform
language automatically handles unknown values during expressions, so that
for example adding a known value to an unknown value automatically produces
an unknown value as the result.

However, there are some situations where unknown values _do_ have a significant
effect:

* The `count` meta-argument for resources cannot be unknown, since it must
  be evaluated during the plan phase to determine how many instances are to
  be created.

* If unknown values are used in the configuration of a data resource, that
  data resource cannot be read during the plan phase and so it will be deferred
  until the apply phase. In this case, the results of the data resource will
  _also_ be unknown values.

* If an unknown value is assigned to an argument inside a `module` block,
  any references to the corresponding input variable within the child module
  will use that unknown value.

* If an unknown value is used in the `value` argument of an output value,
  any references to that output value in the parent module will use that
  unknown value.

* Terraform will attempt to validate that unknown values are of suitable
  types where possible, but incorrect use of such values may not be detected
  until the apply phase, causing the apply to fail.

Unknown values appear in the `terraform plan` output as `(not yet known)`.

## Arithmetic and Logical Operators

An _operator_ is a type of expression that transforms or combines one or more
other expressions. Operators either combine two values in some way to
produce a third result value, or transform a single given value to
produce a single result.

Operators that work on two values place an operator symbol between the two
values, similar to mathematical notation: `1 + 2`. Operators that work on
only one value place an operator symbol before that value, like
`!true`.

The Terraform language has a set of operators for both arithmetic and logic,
which are similar to operators in programming languages such as JavaScript
or Ruby.

When multiple operators are used together in an expression, they are evaluated
in the following order of operations:

1. `!`, `-` (multiplication by `-1`)
1. `*`, `/`, `%`
1. `+`, `-` (subtraction)
1. `>`, `>=`, `<`, `<=`
1. `==`, `!=`
1. `&&`
1. `||`

Parentheses can be used to override the default order of operations. Without
parentheses, higher levels are evaluated first, so `1 + 2 * 3` is interpreted
as `1 + (2 * 3)` and _not_ as `(1 + 2) * 3`.

The different operators can be gathered into a few different groups with
similar behavior, as described below. Each group of operators expects its
given values to be of a particular type. Terraform will attempt to convert
values to the required type automatically, or will produce an error message
if this automatic conversion is not possible.

### Arithmetic Operators

The arithmetic operators all expect number values and produce number values
as results:

* `a + b` returns the result of adding `a` and `b` together.
* `a - b` returns the result of subtracting `b` from `a`.
* `a * b` returns the result of multiplying `a` and `b`.
* `a / b` returns the result of dividing `a` by `b`.
* `a % b` returns the remainder of dividing `a` by `b`. This operator is
  generally useful only when used with whole numbers.
* `-a` returns the result of multiplying `a` by `-1`.

### Equality Operators

The equality operators both take two values of any type and produce boolean
values as results.

* `a == b` returns `true` if `a` and `b` both have the same type and the same
  value, or `false` otherwise.
* `a != b` is the opposite of `a == b`.

### Comparison Operators

The comparison operators all expect number values and produce boolean values
as results.

* `a < b` returns `true` if `a` is less than `b`, or `false` otherwise.
* `a <= b` returns `true` if `a` is less than or equal to `b`, or `false`
  otherwise.
* `a > b` returns `true` if `a` is greater than `b`, or `false` otherwise.
* `a >= b` returns `true` if `a` is greater than or equal to `b`, or `false otherwise.

### Logical Operators

The logical operators all expect bool values and produce bool values as results.

* `a || b` returns `true` if either `a` or `b` is `true`, or `false` if both are `false`.
* `a && b` returns `true` if both `a` and `b` are `true`, or `false` if either one is `false`.
* `!a` returns `true` if `a` is `false`, and `false` if `a` is `true`.

## Conditional Expressions

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

Any of the equality, comparison, and logical operators can be used to define
the condition. The two result values may be of any type, but they must both
be of the _same_ type so that Terraform can determine what type the whole
conditional expression will return without knowing the condition value.

## Function Calls

The Terraform language has a number of
[built-in functions](./functions.html) that can be used
within expressions as another way to transform and combine values. These
are similar to the operators but all follow a common syntax:

```hcl
<FUNCTION NAME>(<ARGUMENT 1>, <ARGUMENT 2>)
```

The function name specifies which function to call. Each defined function
expects a specific number of arguments with specific value types, and returns a
specific value type as a result.

Some functions take an arbitrary number of arguments. For example, the `min`
function takes any amount of number arguments and returns the one that is
numerically smallest:

```hcl
min(55, 3453, 2)
```

### Expanding Function Arguments

If the arguments to pass to a function are available in a list or tuple value,
that value can be _expanded_ into separate arguments. Provide the list value as
an argument and follow it with the `...` symbol:

```hcl
min([55, 2453, 2]...)
```

The expansion symbol is three periods (`...`), not a Unicode ellipsis character
(`…`). Expansion is a special syntax that is only available in function calls.

### Available Functions

For a full list of available functions, see
[the function reference](./functions.html).

## `for` Expressions

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

## Splat Expressions

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

Splat expressions also have another useful effect: if they are applied to
a value that is _not_ a list or tuple then the value is automatically wrapped
in a single-element list before processing. That is, `var.single_object[*].id`
is equivalent to `[var.single_object][*].id`, or effectively
`[var.single_object.id]`. This behavior is not interesting in most cases,
but it is particularly useful when referring to resources that may or may
not have `count` set, and thus may or may not produce a tuple value:

```hcl
aws_instance.example[*].id
```

The above will produce a list of ids whether `aws_instance.example` has
`count` set or not, avoiding the need to revise various other expressions
in the configuration when a particular resource switches to and from
having `count` set.

### Legacy (Attribute-only) Splat Expressions

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

## `dynamic` blocks

Within top-level block constructs like resources, expressions can usually be
used only when assigning a value to an argument using the `name = expression`
form. This covers many uses, but some resource types include repeatable _nested
blocks_ in their arguments, which do not accept expressions:

```hcl
resource "aws_security_group" "example" {
  name = "example" # can use expressions here

  ingress {
    # but the "ingress" block is always a literal block
  }
}
```

You can dynamically construct repeatable nested blocks like `ingress` using a
special `dynamic` block type, which is supported inside `resource`, `data`,
`provider`, and `provisioner` blocks:

```hcl
resource "aws_security_group" "example" {
  name = "example" # can use expressions here

  dynamic "ingress" {
    for_each = var.service_ports
    content {
      from_port = ingress.value
      to_port   = ingress.value
      protocol  = "tcp"
    }
  }
}
```

A `dynamic` block acts much like a `for` expression, but produces nested blocks
instead of a complex typed value. It iterates over a given complex value, and
generates a nested block for each element of that complex value.

- The label of the dynamic block (`"ingress"` in the example above) specifies
  what kind of nested block to generate.
- The `for_each` argument provides the complex value to iterate over.
- The `iterator` argument (optional) sets the name of a temporary variable
  that represents the current element of the complex value. If omitted, the name
  of the variable defaults to the label of the `dynamic` block (`"ingress"` in
  the example above).
- The `labels` argument (optional) is a list of strings that specifies the block
  labels, in order, to use for each generated block. You can use the temporary
  iterator variable in this value.
- The nested `content` block defines the body of each generated block. You can
  use the temporary iterator variable inside this block.

Since the `for_each` argument accepts any collection or structural value,
you can use a `for` expression or splat expression to transform an existing
collection.

The iterator object (`ingress` in the example above) has two attributes:

* `key` is the map key or list element index for the current element. If the
  `for_each` exression produces a _set_ value then `key` is identical to
  `value` and should not be used.
* `value` is the value of the current element.

A `dynamic` block can only generate arguments that belong to the resource type,
data source, provider or provisioner being configured. It is _not_ possible
to generate meta-argument blocks such as `lifecycle` and `provisioner`
blocks, since Terraform must process these before it is safe to evaluate
expressions.

If you need to iterate over combinations of values from multiple collections,
use [`setproduct`](./functions/setproduct.html) to create a single collection
containing all of the combinations.

### Best Practices for `dynamic` Blocks

Overuse of `dynamic` blocks can make configuration hard to read and maintain, so
we recommend using them only when you need to hide details in order to build a
clean user interface for a re-usable module. Always write nested blocks out
literally where possible.

## String Literals

The Terraform language has two different syntaxes for string literals. The
most common is to delimit the string with quote characters (`"`), like
`"hello"`. In quoted strings, the backslash character serves as an escape
sequence, with the following characters selecting the escape behavior:

| Sequence     | Replacement                                                                   |
| ------------ | ----------------------------------------------------------------------------- |
| `\n`         | Newline                                                                       |
| `\r`         | Carriage Return                                                               |
| `\t`         | Tab                                                                           |
| `\"`         | Literal quote (without terminating the string)                                |
| `\\`         | Literal backslash                                                             |
| `\uNNNN`     | Unicode character from the basic multilingual plane (NNNN is four hex digits) |
| `\UNNNNNNNN` | Unicode character from supplementary planes (NNNNNNNN is eight hex digits)    |

The alternative syntax for string literals is the so-called "heredoc" style,
inspired by Unix shell languages. This style allows multi-line strings to
be expressed more clearly by using a custom delimiter word on a line of its
own to close the string:

```hcl
<<EOT
hello
world
EOT
```

The `<<` marker followed by any identifier at the end of a line introduces the
sequence. Terraform then processes the following lines until it finds one that
consists entirely of the identifier given in the introducer. In the above
example, `EOT` is the identifier selected. Any identifier is allowed, but
conventionally this identifier is in all-uppercase and beings with `EO`, meaning
"end of". `EOT` in this case stands for "end of text".

The "heredoc" form shown above requires that the lines following be flush with
the left margin, which can be awkward when an expression is inside an indented
block:

```hcl
block {
  value = <<EOT
hello
world
EOT
}
```

To improve on this, Terraform also accepts an _indented_ heredoc string variant
that is introduced by the `<<-` sequence:

```hcl
block {
  value = <<-EOT
  hello
    world
  EOT
}
```

In this case, Terraform analyses the lines in the sequence to find the one
with the smallest number of leading spaces, and then trims that many spaces
from the beginning of all of the lines, leading to the following result:

```
hello
  world
```

Backslash sequences are not interpreted in a heredoc string expression.
Instead, the backslash character is interpreted literally.

In both quoted and heredoc string expressions, Terraform supports template
sequences that begin with `${` and `%{`. These are described in more detail
in the following section. To include these sequences _literally_ without
beginning a template sequence, double the leading character: `$${` or `%%{`.

## String Templates

Within quoted and heredoc string expressions, the sequences `${` and `%{` begin
_template sequences_. Templates let you directly embed expressions into a string
literal, to dynamically construct strings from other values.

### Interpolation

A `${ ... }` sequence is an _interpolation,_ which evaluates the expression
given between the markers, converts the result to a string if necessary, and
then inserts it into the final string:

```hcl
"Hello, ${var.name}!"
```

In the above example, the named object `var.name` is accessed and its value
inserted into the string, producing a result like "Hello, Juan!".

### Directives

A `%{ ... }` sequence is a _directive_, which allows for conditional
results and iteration over collections, similar to conditional
and `for` expressions.

The following directives are supported:

* The `if <BOOL>`/`else`/`endif` directive chooses between two templates based
  on the value of a bool expression:

    ```hcl
    "Hello, %{ if var.name != "" }${var.name}%{ else }unnamed%{ endif }!"
    ```

    The `else` portion may be omitted, in which case the result is an empty
    string if the condition expression returns `false`.

* The `for <NAME> in <COLLECTION>` / `endfor` directive iterates over the
  elements of a given collection or structural value and evaluates a given
  template once for each element, concatenating the results together:

    ```hcl
    <<EOT
    %{ for ip in aws_instance.example.*.private_ip }
    server ${ip}
    %{ endfor }
    EOT
    ```

    The name given immediately after the `for` keyword is used as a temporary
    variable name which can then be referenced from the nested template.

To allow template directives to be formatted for readability without adding
unwanted spaces and newlines to the result, all template sequences can include
optional _strip markers_ (`~`), immediately after the opening characters or
immediately before the end. When a strip marker is present, the template
sequence consumes all of the literal whitespace (spaces and newlines) either
before the sequence (if the marker appears at the beginning) or after (if the
marker appears at the end):

```hcl
<<EOT
%{ for ip in aws_instance.example.*.private_ip ~}
server ${ip}
%{ endfor ~}
EOT
```

In the above example, the newline after each of the directives is not included
in the output, but the newline after the `server ${ip}` sequence is retained,
causing only one line to be generated for each element:

```
server 10.1.16.154
server 10.1.16.1
server 10.1.16.34
```

When using template directives, we recommend always using the "heredoc" string
literal form and then formatting the template over multiple lines for
readability. Quoted string literals should usually include only interpolation
sequences.
