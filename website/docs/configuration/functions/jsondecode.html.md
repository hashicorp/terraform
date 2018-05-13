---
layout: "functions"
page_title: "jsondecode function"
sidebar_current: "docs-funcs-encoding-jsondecode"
description: |-
  The jsondecode function decodes a JSON string into a representation of its
  value.
---

# `jsondecode` Function

`jsondecode` interprets a given string as JSON, returning a representation
of the result of decoding that string.

The JSON encoding is defined in [RFC 7159](https://tools.ietf.org/html/rfc7159).

This function maps JSON values to
[Terraform language values](/docs/configuration/expressions.html#types-and-values)
in the following way:

| JSON type | Terraform type                                               |
| --------- | ------------------------------------------------------------ |
| String    | `string`                                                     |
| Number    | `number`                                                     |
| Boolean   | `bool`                                                       |
| Object    | `object(...)` with attribute types determined per this table |
| Array     | `tuple(...)` with element types determined per this table    |
| Null      | The Terraform language `null` value                          |

The Terraform language automatic type conversion rules mean that you don't
usually need to worry about exactly what type is produced for a given value,
and can just use the result in an intuitive way.

## Examples

```
> jsondecode("{\"hello\": \"world\"}")
{
  "hello" = "world"
}
> jsondecode("true")
true
```

## Related Functions

* [`jsonencode`](./jsonencode.html) performs the opposite operation, _encoding_
  a value as JSON.
