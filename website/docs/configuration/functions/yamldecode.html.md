---
layout: "functions"
page_title: "yamldecode - Functions - Configuration Language"
sidebar_current: "docs-funcs-encoding-yamldecode"
description: |-
  The yamldecode function decodes a YAML string into a representation of its
  value.
---

# `yamldecode` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`yamldecode` parses a string as a subset of YAML, and produces a representation
of its value.

This function supports a subset of [YAML 1.2](https://yaml.org/spec/1.2/spec.html),
as described below.

This function maps YAML values to
[Terraform language values](../expressions.html#types-and-values)
in the following way:

| YAML type     | Terraform type                                                     |
| ------------- | ------------------------------------------------------------------ |
| `!!str`       | `string`                                                           |
| `!!float`     | `number`                                                           |
| `!!int`       | `number`                                                           |
| `!!bool`      | `bool`                                                             |
| `!!map`       | `object(...)` with attribute types determined per this table       |
| `!!seq`       | `tuple(...)` with element types determined per this table          |
| `!!null`      | The Terraform language `null` value                                |
| `!!timestamp` | `string` in [RFC 3339](https://tools.ietf.org/html/rfc3339) format |
| `!!binary`    | `string` containing base64-encoded representation                  |

The Terraform language automatic type conversion rules mean that you don't
usually need to worry about exactly what type is produced for a given value,
and can just use the result in an intuitive way.

Note though that the mapping above is ambiguous -- several different source
types map to the same target type -- and so round-tripping through `yamldecode`
and then `yamlencode` cannot produce an identical result.

YAML is a complex language and it supports a number of possibilities that the
Terraform language's type system cannot represent. Therefore this YAML decoder
supports only a subset of YAML 1.2, with restrictions including the following:

- Although aliases to earlier anchors are supported, cyclic data structures
  (where a reference to a collection appears inside that collection) are not.
  If `yamldecode` detects such a structure then it will return an error.

- Only the type tags shown in the above table (or equivalent alternative
  representations of those same tags) are supported. Any other tags will
  result in an error.

- Only one YAML document is permitted. If multiple documents are present in
  the given string then this function will return an error.

## Examples

```
> yamldecode("{\"hello\": \"world\"}")
{
  "hello" = "world"
}

> yamldecode("true")
true

> yamldecode("{a: &foo [1, 2, 3], b: *foo}")
{
  "a" = [
    1,
    2,
    3,
  ]
  "b" = [
    1,
    2,
    3,
  ]
}

> yamldecode("{a: &foo [1, *foo, 3]}")

Error: Error in function call

Call to function "yamldecode" failed: cannot refer to anchor "foo" from inside
its own definition.

> yamldecode("{a: !not-supported foo}")

Error: Error in function call

Call to function "yamldecode" failed: unsupported tag "!not-supported".
```

## Related Functions

- [`jsondecode`](./jsondecode.html) is a similar operation using JSON instead
  of YAML.
- [`yamlencode`](./yamlencode.html) performs the opposite operation, _encoding_
  a value as YAML.
