---
layout: "functions"
page_title: "jsonencode - Functions - Configuration Language"
sidebar_current: "docs-funcs-encoding-jsonencode"
description: |-
  The jsonencode function encodes a given value as a JSON string.
---

# `jsonencode` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`jsonencode` encodes a given value to a string using JSON syntax.

The JSON encoding is defined in [RFC 7159](https://tools.ietf.org/html/rfc7159).

This function maps
[Terraform language values](../expressions.html#types-and-values)
to JSON values in the following way:

| Terraform type | JSON type |
| -------------- | --------- |
| `string`       | String    |
| `number`       | Number    |
| `bool`         | Bool      |
| `list(...)`    | Array     |
| `set(...)`     | Array     |
| `tuple(...)`   | Array     |
| `map(...)`     | Object    |
| `object(...)`  | Object    |
| Null value     | `null`    |

Since the JSON format cannot fully represent all of the Terraform language
types, passing the `jsonencode` result to `jsondecode` will not produce an
identical value, but the automatic type conversion rules mean that this is
rarely a problem in practice.

## Examples

```
> jsonencode({"hello"="world"})
{"hello":"world"}
```

## Related Functions

* [`jsondecode`](./jsondecode.html) performs the opposite operation, _decoding_
  a JSON string to obtain its represented value.
