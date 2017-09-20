# HCL JSON Syntax Specification

This is the specification for the JSON serialization for hcl. HCL is a system
for defining configuration languages for applications. The HCL information
model is designed to support multiple concrete syntaxes for configuration,
and this JSON-based format complements [the native syntax](../zclsyntax/spec.md)
by being easy to machine-generate, whereas the native syntax is oriented
towards human authoring and maintenence.

This syntax is defined in terms of JSON as defined in
[RFC7159](https://tools.ietf.org/html/rfc7159). As such it inherits the JSON
grammar as-is, and merely defines a specific methodology for interpreting
JSON constructs into HCL structural elements and expressions.

This mapping is defined such that valid JSON-serialized HCL input can be
produced using standard JSON implementations in various programming languages.
_Parsing_ such JSON has some additional constraints not beyond what is normally
supported by JSON parsers, though adaptations are defined to allow processing
with an off-the-shelf JSON parser with certain caveats, described in later
sections.

## Structural Elements

The HCL language-agnostic information model defines a _body_ as an abstract
container for attribute definitions and child blocks. A body is represented
in JSON as a JSON _object_.

As defined in the language-agnostic model, body processing is done in terms
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

Where the given schema describes a block with a given type name, the object
property with the matching name — if present — serves as a definition of
zero or more blocks of that type.

Processing of child blocks is in terms of nested JSON objects and arrays.
If the schema defines one or more _labels_ for the block type, a nested
object is required for each labelling level, with the object keys serving as
the label values at that level.

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
extra label levels must be represented as objects as in the following examples:

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

Where multiple definitions are included for the same type and labels, the
JSON array is always the value of the property representing the final label,
and contains objects representing block bodies. It is not valid to use an array
at any other point in the block definition structure.

## Expressions

JSON lacks a native expression syntax, so the HCL JSON syntax instead defines
a mapping for each of the JSON value types, including a special mapping for
strings that allows optional use of arbitrary expressions.

### Objects

When interpreted as an expression, a JSON object represents a value of a HCL
object type.

Each property of the JSON object represents an attribute of the HCL object type.
The object type is constructed by enumerating the JSON object properties,
creating for each an attribute whose name exactly matches the property name,
and whose type is the result of recursively applying the expression mapping
rules.

An instance of the constructed object type is then created, whose values
are interpreted by again recursively applying the mapping rules defined in
this section.

It is an error to define the same property name multiple times within a single
JSON object interpreted as an expression.

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

HCL numbers are arbitrary-precision decimal values, so an ideal implementation
of this specification will translate exactly the value given to a number of
corresponding precision.

In practice, off-the-shelf JSON parsers often do not support customizing the
processing of numbers, and instead force processing as 32-bit or 64-bit
floating point values with a potential loss of precision. It is permissable
for a HCL JSON parser to pass on such limitations _if and only if_ the
available precision and other constraints are defined in its documentation.
Calling applications each have differing precision requirements, so calling
applications are free to select an implementation with more limited precision
capabilities should high precision not be required for that application.

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
