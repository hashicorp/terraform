---
layout: "functions"
page_title: "timestamp function"
sidebar_current: "docs-funcs-datetime-timestamp"
description: |-
  The timestamp function returns a string representation of the current date
  and time.
---

# `timestamp` Function

`timestamp` returns the current date and time.

In the Terraform language, timestamps are conventionally represented as
strings using [RFC 3339](https://tools.ietf.org/html/rfc3339)
"Date and Time format" syntax, and so `timestamp` returns a string
in this format.

The result of this function will change every second, so using this function
directly with resource attributes will cause a diff to be detected on every
Terraform run. We do not recommend using this function in resource attributes,
but in rare cases it can be used in conjunction with
[the `ignore_changes` lifecycle meta-argument](./resources.html#ignore_changes)
to take the timestamp only on initial creation of the resource.

Due to the constantly changing return value, the result of this function cannot
be preducted during Terraform's planning phase, and so the timestamp will be
taken only once the plan is being applied.

## Examples

```
> timestamp()
2018-05-13T07:44:12Z
```
