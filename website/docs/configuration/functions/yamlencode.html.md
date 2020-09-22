---
layout: "functions"
page_title: "yamlencode - Functions - Configuration Language"
sidebar_current: "docs-funcs-encoding-yamlencode"
description: |-
  The yamlencode function encodes a given value as a YAML string.
---

# `yamlencode` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`yamlencode` encodes a given value to a string using
[YAML 1.2](https://yaml.org/spec/1.2/spec.html) block syntax.

~> **Warning:** This function is currently **experimental** and its exact
result format may change in future versions of Terraform, based on feedback.
Do not use `yamldecode` to construct a value for any resource argument where
changes to the result would be disruptive. To get a consistent string
representation of a value use [`jsonencode`](./jsonencode.html) instead; its
results are also valid YAML because YAML is a JSON superset.

<!--
    The condition for removing the above warning is that the underlying
    go-cty-yaml module makes a stable release with a commitment to guarantee
    that the representation of particular input will not change without a
    major release. It is not making that commitment at the time of writing to
    allow for responding to user feedback about its output format, since YAML
    is a very flexible format and its initial decisions may prove to be
    sub-optimal when generating YAML intended for specific external consumers.
-->

This function maps
[Terraform language values](../expressions.html#types-and-values)
to YAML tags in the following way:

| Terraform type | YAML type            |
| -------------- | -------------------- |
| `string`       | `!!str`              |
| `number`       | `!!float` or `!!int` |
| `bool`         | `!!bool`             |
| `list(...)`    | `!!seq`              |
| `set(...)`     | `!!seq`              |
| `tuple(...)`   | `!!seq`              |
| `map(...)`     | `!!map`              |
| `object(...)`  | `!!map`              |
| Null value     | `!!null`             |

`yamlencode` uses the implied syntaxes for all of the above types, so it does
not generate explicit YAML tags.

Because the YAML format cannot fully represent all of the Terraform language
types, passing the `yamlencode` result to `yamldecode` will not produce an
identical value, but the Terraform language automatic type conversion rules
mean that this is rarely a problem in practice.

## Examples

```
> yamlencode({"a":"b", "c":"d"})
"a": "b"
"c": "d"

> yamlencode({"foo":[1, 2, 3], "bar": "baz"})
"bar": "baz"
"foo":
- 1
- 2
- 3

> yamlencode({"foo":[1, {"a":"b","c":"d"}, 3], "bar": "baz"})
"bar": "baz"
"foo":
- 1
- "a": "b"
  "c": "d"
- 3
```

`yamlencode` always uses YAML's "block style" for mappings and sequences, unless
the mapping or sequence is empty. To generate flow-style YAML, use
[`jsonencode`](./jsonencode.html) instead: YAML flow-style is a superset
of JSON syntax.

## Related Functions

- [`jsonencode`](./jsonencode.html) is a similar operation using JSON instead
  of YAML.
- [`yamldecode`](./yamldecode.html) performs the opposite operation, _decoding_
  a YAML string to obtain its represented value.
