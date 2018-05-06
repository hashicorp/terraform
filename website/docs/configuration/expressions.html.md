---
layout: "docs"
page_title: "Configuration Expressions"
sidebar_current: "docs-config-expressions"
description: |-
  The Terraform language allows the use of expressions to access data exported
  by resources and to transform and combine that data to produce other values.
---

# Expressions

_Expressions_ are used to refer to or compute values within a configuration.
The simplest expressions are just literal values, like `"hello"` or `5`,
but the Terraform language also allows more complex expressions such as
references to data exported by resources, arithmetic, conditional evaluation,
and a number of built-in functions.

Expressions can be used in a number of places in the Terraform language,
but some contexts place restrictions on which expression constructs are allowed,
such as requiring a literal value of a particular type, or forbidding
references to resource attributes. The other pages in this section describe
the contexts where expressions may be used and which expression features
are allowed in each case.

The following sections describe all of the features of the configuration
syntax.

## Types and Values

The result of an expression is a _value_. All values have a _type_, which
dictates where that value can be used and what transformations can be
applied to it.

A _literal expression_ is an expression that directly represents a particular
constant value.

Expressions are most commonly used to set the values of arguments to resources
and to child modules. In these cases, the argument itself has an expected
type and so the given expression must produce a value of that type. Where
possible, Terraform will automatically convert values from one type to another
in order to produce the expected type. If this isn't possible, Terraform will
produce a type mismatch error and you must update the configuration with
a more suitable expression.

This section describes all of the value types in the Terraform language, and
the literal expression syntax that can be used to create values of each
type.

### Primitive Types

A _primitive_ type is a simple type that isn't made from any other types.
The available primitive types in the Terraform language are:

* `string`: a sequence of Unicode characters representing some text, such
  as `"hello"`.

* `number`: a numeric value. The `number` type can represent both whole
  numbers like `15` and fractional values such as `6.283185`.

* `bool`: either `true` or `false`. `bool` values can be used in conditional
  logic.

The Terraform language will automatically convert `number` and `bool` values
to `string` values when needed, and vice-versa as long as the string contains
a valid representation of a number of boolean value.

* `true` converts to `"true"`, and vice-versa
* `false` converts to `"false"`, and vice-versa
* `15` converts to `"15"`, and vice-versa

### Collection Types

A _collection_ type allows multiple values of another type to be grouped
together as a single value. The type of value _within_ a collection is called
its _element type_, and all collection types must have an element type.

For example, the type `list(string)` means "list of strings", which is a
different type than `list(number)`, a list of numbers. All elements of a
collection must always be of the same type.

The three _collection type kinds_ in the Terraform language are:

* `list(...)`: a sequence of values identified by consecutive whole numbers
  starting with zero.
* `map(...)`: a collection of values where each is identified by a string label.
* `set(...)`: a collection of unique values that do not have any secondary
  identifiers or ordering.

There is no direct syntax for creating collection type values, but the
Terraform language can automatically convert a structural type value (as
defined in the next section) to a similar collection type as long as all
of its elements can be converted to the required element type.

### Structural Types

A _structural_ type is another way to combine multiple values into a single
value, but structural types allow each value to be of a distinct type.

The two _structural type kinds_ in the Terraform language are:

* `object(...)`: has named attributes that each have their own type.
* `tuple(...)`: has a sequence of elements identified by consecutive whole
  numbers starting with zero, where each element has its own type.

An object type value can be created using an object expression:

```hcl
{
  name = "John"
  age  = 52
}
```

The type of the object value created by this expression is
`object({name=string,age=number})`. In most cases it is not important to know
the exact type of an object value, since the Terraform language automatically
checks and converts object types when needed.

Similarly, a tuple type value can be created using a tuple expression:

```hcl
["a", 15, true]
```

The type of the tuple value created by this expression is
`tuple([string, number, bool])`. Tuple values are rarely used directly in
the Terraform language, and are instead usually converted immediately to
list values by converting all of the elements to the same type.

Terraform will automatically convert object values to map values when required,
so usually object and map values can be used interchangably as long as their
contained values are of suitable types.

Likewise, Terraform will automatically convert tuple values to list values
when required, and so tuple and list values can be used interchangably in
most cases too.

Because of these automatic conversions, it is common to not make a strong
distinction between object and map or tuple and list in everyday discussion
of the Terraform language. The Terraform documentation usually discusses the
object and tuple types only in rare cases where it is important to distinguish
them from the map and list types.

## References to Named Objects

A number of different named objects can be accessed from Terraform expressions.
For example, resources are available in expressions as named objects that have
an object value corresponding to the schema of their resource type, accessed by
a dot-separated sequence of names like `aws_instance.example`.

The following named objects are available:

* `TYPE.NAME` is an object representing a
  [managed resource](/docs/configuration/resources.html) of the given type
  and name. If the resource has the `count` argument set, the value is
  a list of objects representing its instances. Any named object that does
  not match one of the other patterns listed below will be interpreted by
  Terraform as a reference to a managed resource.

* `var.NAME` is the value of the
  [input variable](/docs/configuration/variables.html) of the given name.

* `local.NAME` is the value of the
  [local value](/docs/configuration/locals.html) of the given name.

* `module.MOD_NAME.OUTPUT_NAME` is the value of the
  [output value](/docs/configuration/outputs.html) of the given name from the
  [child module call](/docs/configuration/modules.html) of the given name.

* `data.SOURCE.NAME` is an object representing a
  [data resource](/docs/configuration/data-sources.html) of the given data
  source and name. If the resource has the `count` argument set, the value is
  a list of objects representing its instances.

* `path.` is the prefix of a set of named objects that are filesystem
  paths of various kinds:

  * `path.module` is the filesystem path of the module where the expression
    is placed.

  * `path.root` is the filesystem path of the root module of the configuration.

  * `path.cwd` is the filesystem path of the current working directory. In
    normal use of Terraform this is the same as `path.root`, but some advanced
    uses of Terraform run it from a directory other than the root module
    directory, causing these paths to be different.

* `terraform.workspace` is the name of the currently selected
  [workspace](/docs/state/workspaces.html).

Terraform analyses the block bodies of constructs such as resources and module
calls to automatically infer dependencies between objects from the use of
some of these reference types in expressions. For example, an object with an
argument expression that refers to a managed resource creates and implicit
dependency between that object and the resource.

The first name in each of these dot-separated sequence is called a
_variable_, but do not confuse this with the idea of an
[input variable](/docs/configuration/variables.html), which acts as a
customization parameter for a module. Input variables are often referred
to as just "variables" for brevity when the meaning is clear from context,
but due to this other meaning of "variable" in the context of expressions
this documentation page will always refer to input variables by their full
name.

Additional expression variables are available in specific contexts. These are
described in other documentation sections describing those specific features.

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
produce a third result value, or simply transform a single given value to
produce a single result.

Operators that work on two values place an operator symbol between the two
values, similar to mathematical notation: `1 + 2`. Operators that work on
only one value place an operator symbol before that value, like
`!true`.

The Terraform language has a set of operators for both arithmetic and logic,
which are similar to operators in programming languages such as JavaScript
or Ruby.

When multiple operators are used together in an expression, they are evaluated
according to a default order of operations:

| Level | Operators            |
| ----- | -------------------- |
| 6     | `*`, `/`, `%`        |
| 5     | `+`, `-`             |
| 4     | `>`, `>=`, `<`, `<=` |
| 3     | `==`, `!=`           |
| 2     | `&&`                 |
| 1     | `||`                 |

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
* `a * b` returns the result of multiplying `b` and `b`.
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

A _conditional expression_ allows the selection of one of two values based
on whether another bool expression is `true` or `false`.

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
[built-in functions](/docs/configuration/functions.html) that can be used
within expressions as another way to transform and combine values. These
are similar to the operators but all follow a common syntax:

```hcl
function_name(argument1, argument2)
```

The `function_name` specifies which function to call. Each defined function has
a _signature_, which defines how many arguments it expects and what value types
those arguments must have. The signature also defines the type of the result
value for any given set of argument types.

Some functions take an arbitrary number of arguments. For example, the `min`
function takes any amount of number arguments and returns the one that is
numerically smallest:

```hcl
min(55, 3453, 2)
```

If the arguments to pass are available in a list or tuple value, that value
can be _expanded_ into separate arguments using the `...` symbol after that
argument:

```hcl
min([55, 2453, 2]...)
```

For a full list of available functions, see
[the function reference](/docs/configuration/functions.html).

## `for` Expressions

A _`for` expression_ allows you create a structural type value by transforming
another structural or collection type value. Each element in the input value
can correspond to either one or zero values in the result, and an arbitrary
expression can be used to transform each input element into an output element.

For example, if `var.list` is a list of strings then it can be converted to
a list of strings with all-uppercase letters with the following:

```hcl
[for s in var.list: upper(s)]
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
{for s in var.list: s => upper(s)}
```

This expression produces an object whose attributes are the original elements
from `var.list` and their corresponding values are the uppercase versions.

A `for` expression can also include an optional `if` clause to filter elements
from the source collection:

```
[for s in var.list: upper(s) if s != ""]
```

The source value can also be an object or map value, in which case two
temporary variable names can be provided to access the keys and values
respectively:

```
[for k, v in var.map: length(k) + length(v)]
```

Finally, if the result type is an object (using `{` and `}` delimiters) then
the value result expression can be followed by the `...` symbol to group
together results that have a common key:

```
{for s in var.list: substr(s, 0, 1) => s... if s != ""}
```

## Splat Expressions

A _splat expressions_ provides a more concise way to express a common
operation that could otherwise be performed with a `for` expression.

If `var.list` is a list of objects that all have an attribute `id`, then
a list of the ids could be obtained using the following `for` expression:

```
[for o in var.list: o.id]
```

This is equivalent to the following _splat expression_:

```
var.list[*].id
```

The special `[*]` symbol iterates over all of the elements of the list given
to its left and accesses from each one the attribute name given on its
right. A splat expression can also be used to access attributes and indexes
from lists of complex types by extending the sequence of operations to the
right of the symbol:

```
var.list[*].interfaces[0].name
```

The above expression is equivalent to the following `for` expression:

```
[for o in var.list: o.interfaces[0].name]
```

A second variant of the _splat expression_ is the "attribute-only" splat
expression, indicated by the sequence `.*`:

```
var.list.*.interfaces[0].name
```

This form has a subtly different behavior, equivalent to the following
`for` expression:

```
[for o in var.list: o.interfaces][0].name
```

Notice that with the attribute-only splat expression the index operation
`[0]` is applied to the result of the iteration, rather than as part of
the iteration itself.

The standard splat expression `[*]` should be used in most cases, because its
behavior is less surprising. The attribute-only splat expression is supported
only for compatibility with earlier versions of Terraform, and should not be
used in new configurations.

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

## `dynamic` blocks

Expressions can usually be used only when assigning a value to an attribute
argument using the `name = expression` form. This covers many uses, but
some resource types include in their arguments _nested blocks_, which
do not accept expressions:

```hcl
resource "aws_security_group" "example" {
  name = "example" # can use expressions here

  ingress {
    # but the "ingress" block is always a literal block
  }
}
```

To allow nested blocks like `ingress` to be constructed dynamically, a special
block type `dynamic` is supported inside `resource`, `data`, `provider`,
and `provisioner` blocks:

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

A `dynamic` block iterates over a collection or structural value given in its
`for_each` argument, generating a nested block for each element by evaluating
the nested `content` block. When evaluating the block, a temporary variable
is defined that is by default named after the block type being generated,
or `ingress` in this example. An optional additional argument `iterator` can be
used to override the name of the iterator variable.

Since the `for_each` argument accepts any collection or structural value,
you can use a `for` expression or splat expression to transform an existing
collection.

Overuse of `dynamic` blocks can make configuration hard to read and maintain,
so we recommend using this only when a re-usable module is hiding some details.
Avoid creating modules that are just thin wrappers around single resources,
passing through all of the input variables directly to resource arguments.
Always write nested blocks out literally where possible.

A `dynamic` block can only generate arguments that belong to the resource type,
data source, provider or provisioner being configured. It is _not_ possible
to generate meta-argument blocks such as `lifecycle` and `provisioner`
blocks, since Terraform must process these before it is safe to evaluate
expressions.

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
| `\UNNNNNNNN` | Unicode character from supplimentary planes (NNNNNNNN is eight hex digits)    |

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
sequences introduced by `${` and `%{`. These are described in more detail
in the following section. To include these sequences _literally_ without
beginning a template sequence, double the leading character: `$${` or `%%{`.

## String Templates

Within quoted and heredoc string expressions, the sequences `${` and `%{`
begin _template sequences_. Templates allow expressions to be embedded directly
into the string sequence, and thus allow strings to be dynamically constructed
from other values in a concise way.

A `${ ... }` sequence is an _interpolation_, which evaluates the expression
given between the markers, converts the result to a string if necessary, and
then inserts it into the final string:

```hcl
"Hello, ${var.name}!"
```

In the above example, the named object `var.name` is accessed and its value
inserted into the string, producing a result like "Hello, Juan!".

A `%{ ... }` sequence is a _directive_, which allows for conditional
results and iteration over collections, similar to conditional and
and `for` expressions.

The following directives are supported:

* The `if` directive chooses between two templates based on a conditional
  expression:

  ```hcl
  "Hello, %{ if var.name != "" }${var.name}%{ else }unnamed%{ endif }!"
  ```

  The "else" portion may be omitted, in which case the result is an empty
  string if the condition expression returns `false`.

* The `for` directive iterates over each of the elements of a given collection
  or structural value and evaluates a given template once for each element,
  concatenating the results together:

  ```hcl
  <<EOT
  %{ for ip in aws_instance.example.*.private_ip }
  server ${ip}
  %{ endfor }
  EOT
  ```

  The name given immediately after the `for` keyword is used as a temporary
  variable name which can then be referenced from the nested template.

To allow for template directives to be formatted for readability without
introducing unwanted additional spaces and newlines in the result, all
template sequences can include optional _strip markers_ `~` either immediately
after the introducer or immediately before the end. When present, the sequence
consumes all of the literal whitespace (spaces and newlines) either before
or after the sequence:

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
expression form and then formatting the template over multiple lines for
readability. Quoted string literals should usually include only interpolation
sequences.
