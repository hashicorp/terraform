# HCL JSON Syntax Specification

This is the specification for the JSON serialization for hcl. HCL is a system
for defining configuration languages for applications. The HCL information
model is designed to support multiple concrete syntaxes for configuration,
and this JSON-based format complements [the native syntax](../hclsyntax/spec.md)
by being easy to machine-generate, whereas the native syntax is oriented
towards human authoring and maintenence.

This syntax is defined in terms of JSON as defined in
[RFC7159](https://tools.ietf.org/html/rfc7159). As such it inherits the JSON
grammar as-is, and merely defines a specific methodology for interpreting
JSON constructs into HCL structural elements and expressions.

This mapping is defined such that valid JSON-serialized HCL input can be
_produced_ using standard JSON implementations in various programming languages.
_Parsing_ such JSON has some additional constraints not beyond what is normally
supported by JSON parsers, so a specialized parser may be required that
is able to:

* Preserve the relative ordering of properties defined in an object.
* Preserve multiple definitions of the same property name.
* Preserve numeric values to the precision required by the number type
  in [the HCL syntax-agnostic information model](../spec.md).
* Retain source location information for parsed tokens/constructs in order
  to produce good error messages.

## Structural Elements

[The HCL syntax-agnostic information model](../spec.md) defines a _body_ as an
abstract container for attribute definitions and child blocks. A body is
represented in JSON as either a single JSON object or a JSON array of objects.

Body processing is in terms of JSON object properties, visited in the order
they appear in the input. Where a body is represented by a single JSON object,
the properties of that object are visited in order. Where a body is
represented by a JSON array, each of its elements are visited in order and
each element has its properties visited in order. If any element of the array
is not a JSON object then the input is erroneous.

When a body is being processed in the _dynamic attributes_ mode, the allowance
of a JSON array in the previous paragraph does not apply and instead a single
JSON object is always required.

As defined in the language-agnostic model, body processing is in terms
of a schema which provides context for interpreting the body's content. For
JSON bodies, the schema is crucial to allow differentiation of attribute
definitions and block definitions, both of which are represented via object
properties.

The special property name `"//"`, when used in an object representing a HCL
body, is parsed and ignored. A property with this name can be used to
include human-readable comments. (This special property name is _not_
processed in this way for any _other_ HCL constructs that are represented as
JSON objects.)

### Attributes

Where the given schema describes an attribute with a given name, the object
property with the matching name — if present — serves as the attribute's
definition.

When a body is being processed in the _dynamic attributes_ mode, each object
property serves as an attribute definition for the attribute whose name
matches the property name.

The value of an attribute definition property is interpreted as an _expression_,
as described in a later section.

Given a schema that calls for an attribute named "foo", a JSON object like
the following provides a definition for that attribute:

```json
{
  "foo": "bar baz"
}
```

### Blocks

Where the given schema describes a block with a given type name, each object
property with the matching name serves as a definition of zero or more blocks
of that type.

Processing of child blocks is in terms of nested JSON objects and arrays.
If the schema defines one or more _labels_ for the block type, a nested JSON
object or JSON array of objects is required for each labelling level. These
are flattened to a single ordered sequence of object properties using the
same algorithm as for body content as defined above. Each object property
serves as a label value at the corresponding level.

After any labelling levels, the next nested value is either a JSON object
representing a single block body, or a JSON array of JSON objects that each
represent a single block body. Use of an array accommodates the definition
of multiple blocks that have identical type and labels.

Given a schema that calls for a block type named "foo" with no labels, the
following JSON objects are all valid definitions of zero or more blocks of this
type:

```json
{
  "foo": {
    "child_attr": "baz"
  }
}
```

```json
{
  "foo": [
    {
      "child_attr": "baz"
    },
    {
      "child_attr": "boz"
    }
  ]
}
```
```json
{
  "foo": []
}
```

The first of these defines a single child block of type "foo". The second
defines _two_ such blocks. The final example shows a degenerate definition
of zero blocks, though generators should prefer to omit the property entirely
in this scenario.

Given a schema that calls for a block type named "foo" with _two_ labels, the
extra label levels must be represented as objects or arrays of objects as in
the following examples:

```json
{
  "foo": {
    "bar": {
      "baz": {
        "child_attr": "baz"
      },
      "boz": {
        "child_attr": "baz"
      }
    },
    "boz": {
      "baz": {
        "child_attr": "baz"
      },
    }
  }
}
```

```json
{
  "foo": {
    "bar": {
      "baz": {
        "child_attr": "baz"
      },
      "boz": {
        "child_attr": "baz"
      }
    },
    "boz": {
      "baz": [
        {
          "child_attr": "baz"
        },
        {
          "child_attr": "boz"
        }
      ]
    }
  }
}
```

```json
{
  "foo": [
    {
      "bar": {
        "baz": {
          "child_attr": "baz"
        },
        "boz": {
          "child_attr": "baz"
        }
      },
    },
    {
      "bar": {
        "baz": [
          {
            "child_attr": "baz"
          },
          {
            "child_attr": "boz"
          }
        ]
      }
    }
  ]
}
```

```json
{
  "foo": {
    "bar": {
      "baz": {
        "child_attr": "baz"
      },
      "boz": {
        "child_attr": "baz"
      }
    },
    "bar": {
      "baz": [
        {
          "child_attr": "baz"
        },
        {
          "child_attr": "boz"
        }
      ]
    }
  }
}
```

Arrays can be introduced at either the label definition or block body
definition levels to define multiple definitions of the same block type
or labels while preserving order.

A JSON HCL parser _must_ support duplicate definitions of the same property
name within a single object, preserving all of them and the relative ordering
between them. The array-based forms are also required so that JSON HCL
configurations can be produced with JSON producing libraries that are not
able to preserve property definition order and multiple definitions of
the same property.

## Expressions

JSON lacks a native expression syntax, so the HCL JSON syntax instead defines
a mapping for each of the JSON value types, including a special mapping for
strings that allows optional use of arbitrary expressions.

### Objects

When interpreted as an expression, a JSON object represents a value of a HCL
object type.

Each property of the JSON object represents an attribute of the HCL object type.
The property name string given in the JSON input is interpreted as a string
expression as described below, and its result is converted to string as defined
by the syntax-agnostic information model. If such a conversion is not possible,
an error is produced and evaluation fails.

An instance of the constructed object type is then created, whose values
are interpreted by again recursively applying the mapping rules defined in
this section to each of the property values.

If any evaluated property name strings produce null values, an error is
produced and evaluation fails. If any produce _unknown_ values, the _entire
object's_ result is an unknown value of the dynamic pseudo-type, signalling
that the type of the object cannot be determined.

It is an error to define the same property name multiple times within a single
JSON object interpreted as an expression. In full expression mode, this
constraint applies to the name expression results after conversion to string,
rather than the raw string that may contain interpolation expressions.

### Arrays

When interpreted as an expression, a JSON array represents a value of a HCL
tuple type.

Each element of the JSON array represents an element of the HCL tuple type.
The tuple type is constructed by enumerationg the JSON array elements, creating
for each an element whose type is the result of recursively applying the
expression mapping rules. Correspondance is preserved between the array element
indices and the tuple element indices.

An instance of the constructed tuple type is then created, whose values are
interpreted by again recursively applying the mapping rules defined in this
section.

### Numbers

When interpreted as an expression, a JSON number represents a HCL number value.

HCL numbers are arbitrary-precision decimal values, so a JSON HCL parser must
be able to translate exactly the value given to a number of corresponding
precision, within the constraints set by the HCL syntax-agnostic information
model.

In practice, off-the-shelf JSON serializers often do not support customizing the
processing of numbers, and instead force processing as 32-bit or 64-bit
floating point values.

A _producer_ of JSON HCL that uses such a serializer can provide numeric values
as JSON strings where they have precision too great for representation in the
serializer's chosen numeric type in situations where the result will be
converted to number (using the standard conversion rules) by a calling
application.

Alternatively, for expressions that are evaluated in full expression mode an
embedded template interpolation can be used to faithfully represent a number,
such as `"${1e150}"`, which will then be evaluated by the underlying HCL native
syntax expression evaluator.

### Boolean Values

The JSON boolean values `true` and `false`, when interpreted as expressions,
represent the corresponding HCL boolean values.

### The Null Value

The JSON value `null`, when interpreted as an expression, represents a
HCL null value of the dynamic pseudo-type.

### Strings

When intepreted as an expression, a JSON string may be interpreted in one of
two ways depending on the evaluation mode.

If evaluating in literal-only mode (as defined by the syntax-agnostic
information model) the literal string is intepreted directly as a HCL string
value, by directly using the exact sequence of unicode characters represented.
Template interpolations and directives MUST NOT be processed in this mode,
allowing any characters that appear as introduction sequences to pass through
literally:

```json
"Hello world! Template sequences like ${ are not intepreted here."
```

When evaluating in full expression mode (again, as defined by the syntax-
agnostic information model) the literal string is instead interpreted as a
_standalone template_ in the HCL Native Syntax. The expression evaluation
result is then the direct result of evaluating that template with the current
variable scope and function table.

```json
"Hello, ${name}! Template sequences are interpreted in full expression mode."
```

In particular the _Template Interpolation Unwrapping_ requirement from the
HCL native syntax specification must be implemented, allowing the use of
single-interpolation templates to represent expressions that would not
otherwise be representable in JSON, such as the following example where
the result must be a number, rather than a string representation of a number:

```json
"${ a + b }"
```

## Static Analysis

The HCL static analysis operations are implemented for JSON values that
represent expressions, as described in the following sections.

Due to the limited expressive power of the JSON syntax alone, use of these
static analyses functions rather than normal expression evaluation is used
as additional context for how a JSON value is to be interpreted, which means
that static analyses can result in a different interpretation of a given
expression than normal evaluation.

### Static List

An expression interpreted as a static list must be a JSON array. Each of the
values in the array is interpreted as an expression and returned.

### Static Map

An expression interpreted as a static map must be a JSON object. Each of the
key/value pairs in the object is presented as a pair of expressions. Since
object property names are always strings, evaluating the key expression with
a non-`nil` evaluation context will evaluate any template sequences given
in the property name.

### Static Call

An expression interpreted as a static call must be a string. The content of
the string is interpreted as a native syntax expression (not a _template_,
unlike normal evaluation) and then the static call analysis is delegated to
that expression.

If the original expression is not a string or its contents cannot be parsed
as a native syntax expression then static call analysis is not supported.

### Static Traversal

An expression interpreted as a static traversal must be a string. The content
of the string is interpreted as a native syntax expression (not a _template_,
unlike normal evaluation) and then static traversal analysis is delegated
to that expression.

If the original expression is not a string or its contents cannot be parsed
as a native syntax expression then static call analysis is not supported.

