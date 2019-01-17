---
layout: "docs"
page_title: "Type Constraints - Configuration Language"
sidebar_current: "docs-config-types"
description: |-
  Terraform module authors and provider developers can use detailed type
  constraints to validate the inputs of their modules and resources.
---

# Type Constraints

-> **Note:** This page is about Terraform 0.12 and later, and documents a
feature that did not exist in older versions. For other information about
Terraform 0.11 and earlier, see
[0.11 Configuration Language](../configuration-0-11/index.html).

Terraform module authors and provider developers can use detailed type
constraints to validate user-provided values for their input variables and
resource arguments. This requires some additional knowledge about Terraform's
type system, but allows you to build a more resilient user interface for your
modules and resources.

## Type Keywords and Constructors

Type constraints are expressed using a mixture of _type keywords_ and
function-like constructs called _type constructors._

* Type keywords are unquoted symbols that represent a static type.
* Type constructors are unquoted symbols followed by a pair of
  parentheses, which contain an argument that specifies more information about
  the type. Without its argument, a type constructor does not fully
  represent a type; instead, it represents a _kind_ of similar types.

Type constraints look like other kinds of Terraform
[expressions](./expressions.html), but are a special syntax. Within the
Terraform language, they are only valid in the `type` argument of an
[input variable](./variables.html).

## Primitive Types

A _primitive_ type is a simple type that isn't made from any other types. All
primitive types in Terraform are represented by a type keyword. The available
primitive types are:

* `string`: a sequence of Unicode characters representing some text, such
  as `"hello"`.
* `number`: a numeric value. The `number` type can represent both whole
  numbers like `15` and fractional values such as `6.283185`.
* `bool`: either `true` or `false`. `bool` values can be used in conditional
  logic.

### Conversion of Primitive Types

The Terraform language will automatically convert `number` and `bool` values
to `string` values when needed, and vice-versa as long as the string contains
a valid representation of a number or boolean value.

* `true` converts to `"true"`, and vice-versa
* `false` converts to `"false"`, and vice-versa
* `15` converts to `"15"`, and vice-versa

## The "Any" Type

The type keyword `any` is a special type constraint that accepts any value.

## Complex Types

A _complex_ type is a type that groups multiple values into a single value.
Complex types are represented by type constructors, but several of them
also have shorthand keyword versions.

There are two categories of complex types: collection types (for grouping
similar values), and structural types (for grouping potentially dissimilar
values).

### Collection Types

A _collection_ type allows multiple values of _one_ other type to be grouped
together as a single value. The type of value _within_ a collection is called
its _element type._ All collection types must have an element type, which is
provided as the argument to their constructor.

For example, the type `list(string)` means "list of strings", which is a
different type than `list(number)`, a list of numbers. All elements of a
collection must always be of the same type.

The three kinds of collection type in the Terraform language are:

* `list(...)`: a sequence of values identified by consecutive whole numbers
  starting with zero.

    The keyword `list` is a shorthand for `list(any)`, which accepts any
    element type as long as every element is the same type. This is for
    compatibility with older configurations; for new code, we recommend using
    the full form.
* `map(...)`: a collection of values where each is identified by a string label.

    The keyword `map` is a shorthand for `map(any)`, which accepts any
    element type as long as every element is the same type. This is for
    compatibility with older configurations; for new code, we recommend using
    the full form.
* `set(...)`: a collection of unique values that do not have any secondary
  identifiers or ordering.

### Structural Types

A _structural_ type allows multiple values of _several distinct types_ to be
grouped together as a single value. Structural types require a _schema_ as an
argument, to specify which types are allowed for which elements.

The two kinds of structural type in the Terraform language are:

* `object(...)`: a collection of named attributes that each have their own type.

    The schema for object types is `{ <KEY> = <TYPE>, <KEY> = <TYPE>, ... }` — a
    pair of curly braces containing a comma-separated series of `<KEY> = <TYPE>`
    pairs. Values that match the object type must contain _all_ of the specified
    keys, and the value for each key must match its specified type. (Values with
    _additional_ keys can still match an object type, but the extra attributes
    are discarded during type conversion.)
* `tuple(...)`: a sequence of elements identified by consecutive whole
  numbers starting with zero, where each element has its own type.

    The schema for tuple types is `[<TYPE>, <TYPE>, ...]` — a pair of square
    brackets containing a comma-separated series of types. Values that match the
    tuple type must have _exactly_ the same number of elements (no more and no
    fewer), and the value in each position must match the specified type for
    that position.

For example: an object type of `object({ name=string, age=number })` would match
a value like the following:

```hcl
{
  name = "John"
  age  = 52
}
```

Also, an object type of `object({ id=string, cidr_block=string })` would match
the object produced by a reference to an `aws_vpc` resource, like
`aws_vpc.example_vpc`; although the resource has additional attributes, they
would be discarded during type conversion.

Finally, a tuple type of `tuple([string, number, bool])` would match a value
like the following:

```hcl
["a", 15, true]
```

### Complex Type Literals

The Terraform language has literal expressions for creating tuple and object
values, which are described in
[Expressions: Literal Expressions](./expressions.html#literal-expressions) as
"list/tuple" literals and "map/object" literals, respectively.

Terraform does _not_ provide any way to directly represent lists, maps, or sets.
However, due to the automatic conversion of complex types (described below), the
difference between similar complex types is almost never relevant to a normal
user, and most of the Terraform documentation conflates lists with tuples and
maps with objects. The distinctions are only useful when restricting input
values for a module or resource.

### Conversion of Complex Types

Similar kinds of complex types (list/tuple/set and map/object) can usually be
used interchangeably within the Terraform language, and most of Terraform's
documentation glosses over the differences between the kinds of complex type.
This is due to two conversion behaviors:

* Whenever possible, Terraform converts values between similar kinds of complex
  types if the provided value is not the exact type requested. "Similar kinds"
  is defined as follows:
    * Objects and maps are similar.
        * A map (or a larger object) can be converted to an object if it has
          _at least_ the keys required by the object schema. Any additional
          attributes are discarded during conversion, which means map -> object
          -> map conversions can be lossy.
    * Tuples and lists are similar.
        * A list can only be converted to a tuple if it has _exactly_ the
          required number of elements.
    * Sets are _almost_ similar to both tuples and lists:
        * When a list or tuple is converted to a set, duplicate values are
          discarded and the ordering of elements is lost.
        * When a `set` is converted to a list or tuple, the elements will be
          in an arbitrary order. If the set's elements were strings, they will
          be in lexicographical order; sets of other element types do not
          guarantee any particular order of elements.
* Whenever possible, Terraform converts _element values_ within a complex type,
  either by converting complex-typed elements recursively or as described above
  in [Conversion of Primitive Types](#conversion-of-primitive-types).

For example: if a module argument requires a value of type `list(string)` and a
user provides the tuple `["a", 15, true]`, Terraform will internally transform
the value to `["a", "15", "true"]` by converting the elements to the required
`string` element type. Later, if the module uses those elements to set different
resource arguments that require a string, a number, and a bool (respectively),
Terraform will automatically convert the second and third strings back to the
required types at that time, since they contain valid representations of a
number and a bool.

On the other hand, automatic conversion will fail if the provided value
(including any of its element values) is incompatible with the required type. If
an argument requires a type of `map(string)` and a user provides the object
`{name = ["Kristy", "Claudia", "Mary Anne", "Stacey"], age = 12}`, Terraform
will raise a type mismatch error, since a tuple cannot be converted to a string.
