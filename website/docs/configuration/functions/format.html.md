---
layout: "functions"
page_title: "format - Functions - Configuration Language"
sidebar_current: "docs-funcs-string-format-x"
description: |-
  The format function produces a string by formatting a number of other values
  according to a specification string.
---

# `format` Function

`format` produces a string by formatting a number of other values according
to a specification string. It is similar to the `printf` function in C, and
other similar functions in other programming languages.

```hcl
format(spec, values...)
```

## Examples

```
> format("Hello, %s!", "Ander")
Hello, Ander!
> format("There are %d lights", 4)
There are 4 lights
```

Simple format verbs like `%s` and `%d` behave similarly to template
interpolation syntax, which is often more readable:

```
> format("Hello, %s!", var.name)
Hello, Valentina!
> "Hello, ${var.name}!"
Hello, Valentina!
```

The `format` function is therefore more useful when you use more complex format
specifications, as described in the following section.

## Specification Syntax

The specification is a string that includes formatting verbs that are introduced
with the `%` character. The function call must then have one additional argument
for each verb sequence in the specification. The verbs are matched with
consecutive arguments and formatted as directed, as long as each given argument
is convertible to the type required by the format verb.

The specification may contain the following verbs:

| Verb  | Result                                                                                    |
| ----- | ----------------------------------------------------------------------------------------- |
| `%%`  | Literal percent sign, consuming no value.                                                 |
| `%v`  | Default formatting based on the value type, as described below.                           |
| `%#v` | JSON serialization of the value, as with `jsonencode`.                                    |
| `%t`  | Convert to boolean and produce `true` or `false`.                                         |
| `%b`  | Convert to integer number and produce binary representation.                              |
| `%d`  | Convert to integer number and produce decimal representation.                             |
| `%o`  | Convert to integer number and produce octal representation.                               |
| `%x`  | Convert to integer number and produce hexadecimal representation with lowercase letters.  |
| `%X`  | Like `%x`, but use uppercase letters.                                                     |
| `%e`  | Convert to number and produce scientific notation, like `-1.234456e+78`.                  |
| `%E`  | Like `%e`, but use an uppercase `E` to introduce the exponent.                            |
| `%f`  | Convert to number and produce decimal fraction notation with no exponent, like `123.456`. |
| `%g`  | Like `%e` for large exponents or like `%f` otherwise.                                     |
| `%G`  | Like `%E` for large exponents or like `%f` otherwise.                                     |
| `%s`  | Convert to string and insert the string's characters.                                     |
| `%q`  | Convert to string and produce a JSON quoted string representation.                        |

When `%v` is used, one of the following format verbs is chosen based on the value type:

| Type      | Verb  |
| --------- | ----- |
| `string`  | `%s`  |
| `number`  | `%g`  |
| `bool`    | `%t`  |
| any other | `%#v` |

Null values produce the string `null` if formatted with `%v` or `%#v`, and
cause an error for other verbs.

A width modifier can be included with an optional decimal number immediately
preceding the verb letter, to specify how many characters will be used to
represent the value. Precision can be specified after the (optional) width
with a period (`.`) followed by a decimal number. If width or precision are
omitted then default values are selected based on the given value. For example:

| Sequence | Result                       |
| -------- | ---------------------------- |
| `%f`     | Default width and precision. |
| `%9f`    | Width 9, default precision.  |
| `%.2f`   | Default width, precision 2.  |
| `%9.2f`  | Width 9, precision 2.        |

The following additional symbols can be used immediately after the `%` symbol
to set additoinal flags:

| Symbol | Result                                                         |
| ------ | -------------------------------------------------------------- |
| space  | Leave a space where the sign would be if a number is positive. |
| `+`    | Show the sign of a number even if it is positive.              |
| `-`    | Pad the width with spaces on the left rather than the right.   |
| `0`    | Pad the width with zeros rather than spaces.                   |

By default, `%` sequences consume successive arguments starting with the first.
Introducing a `[n]` sequence immediately before the verb letter, where `n` is a
decimal integer, explicitly chooses a particular value argument by its
one-based index. Subsequent calls without an explicit index will then proceed
with `n`+1, `n`+2, etc.

The function produces an error if the format string requests an impossible
conversion or access more arguments than are given. An error is produced also
for an unsupported format verb.

## Related Functions

* [`formatdate`](./formatdate.html) is a specialized formatting function for
  human-readable timestamps.
* [`formatlist`](./formatlist.html) uses the same specification syntax to
  produce a list of strings.
